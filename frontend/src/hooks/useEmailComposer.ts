import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { useEmailSenders, useSendCustomEmail } from '@/hooks/api/useEmails';
import { buildEmailPreviewDocument, parseEmailRecipients, plainTextFromHtml } from '@/components/email/email-composer-utils';

export function useEmailComposer() {
  const editorRef = useRef<HTMLDivElement>(null);
  const [from, setFrom] = useState('');
  const [recipients, setRecipients] = useState('');
  const [cc, setCc] = useState('');
  const [bcc, setBcc] = useState('');
  const [replyTo, setReplyTo] = useState('');
  const [subject, setSubject] = useState('');
  const [richHtml, setRichHtml] = useState('');
  const [sourceHtml, setSourceHtml] = useState('');
  const [mode, setMode] = useState('rich');
  const [attachments, setAttachments] = useState<File[]>([]);
  const [error, setError] = useState('');
  const { data: senders = [], isLoading: isLoadingSenders } = useEmailSenders();
  const sendEmail = useSendCustomEmail();

  useEffect(() => {
    if (!from && senders[0]) setFrom(senders[0].address);
  }, [from, senders]);

  const htmlBody = mode === 'html' ? sourceHtml.trim() : richHtml.trim();
  const previewDocument = useMemo(() => buildEmailPreviewDocument(htmlBody), [htmlBody]);
  const canSend = Boolean(from && parseEmailRecipients(recipients).length > 0 && subject.trim() && htmlBody);

  const handleEditorInput = useCallback(() => {
    setRichHtml(editorRef.current?.innerHTML ?? '');
    setError('');
  }, []);

  const handleAttachmentChange = useCallback((event: React.ChangeEvent<HTMLInputElement>) => {
    setAttachments(Array.from(event.target.files ?? []));
  }, []);

  const handleSend = useCallback(async () => {
    if (!canSend) {
      setError('Vui lòng chọn địa chỉ gửi, nhập người nhận, tiêu đề và nội dung email.');
      return;
    }
    try {
      await sendEmail.mutateAsync({
        from,
        recipients: parseEmailRecipients(recipients),
        cc: parseEmailRecipients(cc),
        bcc: parseEmailRecipients(bcc),
        replyTo: replyTo.trim() || undefined,
        subject: subject.trim(),
        htmlBody,
        textBody: plainTextFromHtml(htmlBody),
        attachments,
      });
      setRecipients('');
      setCc('');
      setBcc('');
      setReplyTo('');
      setSubject('');
      setRichHtml('');
      setSourceHtml('');
      setAttachments([]);
      if (editorRef.current) editorRef.current.innerHTML = '';
    } catch {
      // The shared mutation handler displays the API error and keeps the draft intact.
    }
  }, [attachments, bcc, canSend, cc, from, htmlBody, recipients, replyTo, sendEmail, subject]);

  return {
    attachments,
    bcc,
    canSend,
    cc,
    editorRef,
    error,
    from,
    handleAttachmentChange,
    handleEditorInput,
    handleSend,
    htmlBody,
    isLoadingSenders,
    mode,
    previewDocument,
    recipients,
    replyTo,
    sendEmail,
    senders,
    setAttachments,
    setBcc,
    setCc,
    setError,
    setFrom,
    setMode,
    setRecipients,
    setReplyTo,
    setSourceHtml,
    setSubject,
    sourceHtml,
    subject,
  };
}
