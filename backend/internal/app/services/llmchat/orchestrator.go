package llmchat

import (
	"context"
	"errors"
	"strings"
	"time"

	"api-server/internal/domain"
	llmport "api-server/internal/infra/llm"
	"api-server/internal/pkg/clock"
)

// Turn execution. Fast contract: flash model only, hard per-turn deadline over
// model + tools end to end, bounded context (last N turns), capped tool
// rounds, and a per-turn timing row so the p95 budget is measurable in
// production. A var so tests can shorten it.
// Vision turns are multi-round (each round re-sends the image for context:
// ~20s per provider call + tool execution). Text-only turns answer in <10s.
var TurnTimeout = 120 * time.Second

const (
	// ContextMaxMessages bounds the prompt tail. Running-summary generation
	// for longer threads is a follow-up (noted in the phase report).
	ContextMaxMessages = 12
	// MaxToolRounds caps model round trips per turn. Tool turns are ≥2
	// serialized round trips; beyond this the turn must land.
	MaxToolRounds = 2
	// MaxCompletionTokens caps model output — answers are ≤4 sentences.
	MaxCompletionTokens = 700
	// MaxMessageContent is the stored/wrapped message content bound.
	MaxMessageContent = 4000
)

var turnTemperature = 0.2

// TurnResult is the outcome of one message turn.
type TurnResult struct {
	AssistantMessage *domain.ChatMessage
	ToolCalls        []*domain.ChatToolCall
	Timing           *domain.ChatTurnTiming
}

// runTurn executes one full assistant turn: prompt assembly, model loop with
// parallel tool dispatch, persistence, and timing. The caller has already
// persisted the user message and passed ownership + rate checks.
func (s *Service) runTurn(ctx context.Context, caller Caller, conv *domain.ChatConversation, userText string, images []llmport.ContentPart) (*TurnResult, error) {
	turnCtx, cancel := context.WithTimeout(ctx, TurnTimeout)
	defer cancel()
	start := time.Now()

	timing := &domain.ChatTurnTiming{
		ConversationID: conv.ID,
		UserID:         caller.UserID,
		Role:           caller.Role,
		Model:          llmport.ModelFlash,
	}

	msgs, err := s.buildPromptMessages(turnCtx, caller, conv.ID, userText, images)
	if err != nil {
		s.recordTiming(turnCtx, timing, start, false)
		return nil, err
	}

	manifest := s.registry.Manifest(caller.Role)
	var auditRowIDs []uint

	for round := 0; ; round++ {
		timing.RoundTrips++

		modelStart := time.Now()
		chatModel := llmport.ModelFlash
		if s.models != nil {
			if m := s.models.ChatModel(turnCtx); m != "" {
				chatModel = m
			}
		}
		resp, err := s.llm.Chat(turnCtx, caller.UserID, llmport.ChatRequest{
			Model:       chatModel,
			Messages:    msgs,
			Tools:       manifest,
			ToolChoice:  "auto",
			MaxTokens:   MaxCompletionTokens,
			Temperature: &turnTemperature,
		})
		timing.ModelMS += time.Since(modelStart).Milliseconds()
		if err != nil {
			s.recordTiming(turnCtx, timing, start, isTimeout(turnCtx, err))
			return nil, wrapTurnError(turnCtx, err)
		}

		content, toolCalls := firstChoice(resp)

		// Vision cost control: the image rides only the FIRST model call —
		// by now the model has read it and incorporated it into its tool
		// choice. Later rounds carry tool results, not re-visioned pixels,
		// cutting each subsequent round from ~20s to text-only latency.
		if round == 0 {
			msgs = stripImageParts(msgs)
		}

		finalRound := round >= MaxToolRounds

		if len(toolCalls) == 0 || finalRound {
			if strings.TrimSpace(content) == "" {
				// The model produced no usable answer — land on the designed
				// fallback instead of an empty bubble.
				content = FallbackUnsupportedAction(caller.Role, FallbackTopic(userText))
			}
			asstMsg := &domain.ChatMessage{
				ConversationID: conv.ID,
				UserID:         caller.UserID,
				Role:           domain.ChatMessageRoleAssistant,
				Content:        truncateRunes(content, MaxMessageContent),
			}
			if err := s.msgRepo.Create(turnCtx, asstMsg); err != nil {
				s.recordTiming(turnCtx, timing, start, false)
				return nil, err
			}
			if err := s.convRepo.RecordMessage(turnCtx, conv.ID, clock.Now()); err != nil {
				s.logger.Warn("llmchat: failed to refresh conversation counters", "error", err)
			}
			s.attachAuditRows(turnCtx, auditRowIDs, asstMsg.ID)
			s.recordTiming(turnCtx, timing, start, false)

			result := &TurnResult{AssistantMessage: asstMsg, Timing: timing}
			result.ToolCalls, _ = s.toolCalls.ListByMessage(turnCtx, asstMsg.ID)
			if len(result.ToolCalls) == 0 {
				result.ToolCalls = s.recentToolCalls(turnCtx, conv.ID)
			}
			return result, nil
		}

		// Tool round: execute all calls in parallel (bounded by the turn
		// deadline), then feed results back as untrusted data.
		toolStart := time.Now()
		results, err := s.executor.Execute(turnCtx, caller, conv.ID, toolCalls, nil)
		timing.ToolMS += time.Since(toolStart).Milliseconds()
		if err != nil {
			s.recordTiming(turnCtx, timing, start, isTimeout(turnCtx, err))
			return nil, wrapTurnError(turnCtx, err)
		}
		timing.ToolCalls += len(results)

		msgs = append(msgs, llmport.Message{Role: llmport.RoleAssistant, ToolCalls: toolCalls})
		for _, r := range results {
			if r.AuditRowID != 0 {
				auditRowIDs = append(auditRowIDs, r.AuditRowID)
			}
			msgs = append(msgs, llmport.Message{
				Role:       llmport.RoleTool,
				ToolCallID: r.ToolCallID,
				Content:    []llmport.ContentPart{llmport.TextPart(WrapToolResult(r.Name, r.Text))},
			})
		}
	}
}

// buildPromptMessages assembles the model request: byte-stable system prompt +
// the bounded conversation tail, every externally sourced string wrapped as
// untrusted data.
func (s *Service) buildPromptMessages(ctx context.Context, caller Caller, conversationID uint, userText string, images []llmport.ContentPart) ([]llmport.Message, error) {
	tail, err := s.msgRepo.RecentTail(ctx, conversationID, ContextMaxMessages)
	if err != nil {
		return nil, err
	}

	msgs := []llmport.Message{{
		Role:    llmport.RoleSystem,
		Content: []llmport.ContentPart{llmport.TextPart(BuildSystemPrompt(caller.Role))},
	}}
	// Runtime date anchor lives in the per-request tail, NOT the frozen system
	// prompt, so the byte-stable prefix keeps its prompt-cache hit. Without it
	// the model resolves "năm nay" by guessing — the observed failure where it
	// claimed a year it had never queried.
	msgs = append(msgs, llmport.Message{
		Role:    llmport.RoleSystem,
		Content: []llmport.ContentPart{llmport.TextPart(RuntimeDateContext())},
	})
	for _, m := range tail {
		text := truncateRunes(m.Content, MaxMessageContent)
		if m.Role == domain.ChatMessageRoleUser {
			msgs = append(msgs, llmport.Message{
				Role:    llmport.RoleUser,
				Content: []llmport.ContentPart{llmport.TextPart(WrapUserContent(text))},
			})
		} else {
			msgs = append(msgs, llmport.Message{
				Role:    llmport.RoleAssistant,
				Content: []llmport.ContentPart{llmport.TextPart(WrapAssistantTurn(text))},
			})
		}
	}
	currentContent := append(
		[]llmport.ContentPart{llmport.TextPart(WrapUserContent(truncateRunes(userText, MaxMessageContent)))},
		images...,
	)
	msgs = append(msgs, llmport.Message{
		Role:    llmport.RoleUser,
		Content: currentContent,
	})
	return msgs, nil
}

// firstChoice extracts the first choice's content and tool calls. A response
// with no choices degrades to empty content (the caller lands on fallback).
func firstChoice(resp *llmport.ChatResponse) (string, []llmport.ToolCall) {
	if resp == nil || len(resp.Choices) == 0 {
		return "", nil
	}
	msg := resp.Choices[0].Message
	return msg.Content, msg.ToolCalls
}

// wrapTurnError normalises turn failures: deadline expiry becomes the typed
// Vietnamese ErrTurnTimeout; LLM provider errors pass through unchanged (their
// messages are already designed Vietnamese copy).
func wrapTurnError(ctx context.Context, err error) error {
	if isTimeout(ctx, err) {
		return ErrTurnTimeout
	}
	return err
}

func isTimeout(ctx context.Context, err error) bool {
	return errors.Is(err, context.DeadlineExceeded) ||
		(errors.Is(err, context.Canceled) && ctx.Err() != nil) ||
		errors.Is(err, ErrTurnTimeout)
}

// recordTiming persists the latency row (best effort — instrumentation must
// never take a turn down). Failed and timed-out turns are recorded too, so p95
// reflects real tails.
func (s *Service) recordTiming(ctx context.Context, timing *domain.ChatTurnTiming, start time.Time, timedOut bool) {
	timing.TimedOut = timedOut
	timing.DurationMS = time.Since(start).Milliseconds()
	// The turn context may already be expired; use a fresh one for the write.
	writeCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
	defer cancel()
	if err := s.timingRepo.Create(writeCtx, timing); err != nil {
		s.logger.Warn("llmchat: failed to persist turn timing", "error", err)
	}
}

// attachAuditRows links executed tool-call audit rows to the assistant message
// they fed (best effort).
func (s *Service) attachAuditRows(ctx context.Context, ids []uint, messageID uint) {
	if len(ids) == 0 || s.toolCalls == nil {
		return
	}
	if err := s.toolCalls.AttachMessage(ctx, ids, messageID); err != nil {
		s.logger.Warn("llmchat: failed to link tool calls to message", "error", err)
	}
}

// recentToolCalls returns the conversation's most recent tool calls — the
// fallback view when per-message linkage rows are unavailable.
func (s *Service) recentToolCalls(ctx context.Context, conversationID uint) []*domain.ChatToolCall {
	if s.toolCalls == nil {
		return nil
	}
	calls, err := s.toolCalls.ListByConversation(ctx, conversationID, 20)
	if err != nil {
		s.logger.Warn("llmchat: failed to read tool calls", "error", err)
		return nil
	}
	return calls
}

// stripImageParts removes image content parts from every message (text-only
// replay for tool-result rounds).
func stripImageParts(msgs []llmport.Message) []llmport.Message {
	out := make([]llmport.Message, 0, len(msgs))
	for _, m := range msgs {
		hasImage := false
		for _, part := range m.Content {
			if part.Type == llmport.ContentPartImageURL {
				hasImage = true
				break
			}
		}
		if !hasImage {
			out = append(out, m)
			continue
		}
		texts := make([]llmport.ContentPart, 0, len(m.Content))
		for _, part := range m.Content {
			if part.Type == llmport.ContentPartText {
				texts = append(texts, part)
			}
		}
		m.Content = texts
		out = append(out, m)
	}
	return out
}
