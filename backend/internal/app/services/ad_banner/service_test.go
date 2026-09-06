package ad_banner

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"

	"api-server/internal/domain"
)

// --- fakes ---------------------------------------------------------------

// callLog records cross-fake call ordering (e.g. cache invalidated only
// after the repo mutation succeeded).
type callLog struct {
	events []string
}

func (l *callLog) add(event string) { l.events = append(l.events, event) }

func (l *callLog) hasAfter(candidate, after string) bool {
	candidateIdx, afterIdx := -1, -1
	for i, e := range l.events {
		if e == candidate && candidateIdx == -1 {
			candidateIdx = i
		}
		if e == after && afterIdx == -1 {
			afterIdx = i
		}
	}
	return candidateIdx > afterIdx && candidateIdx != -1 && afterIdx != -1
}

type fakeBannerRepo struct {
	log       *callLog
	banners   []*domain.AdBanner
	nextID    uint
	listCalls int
	clicks    []domain.AdBannerCTAClick
}

func (f *fakeBannerRepo) Create(_ context.Context, banner *domain.AdBanner) error {
	f.nextID++
	banner.ID = f.nextID
	f.banners = append(f.banners, banner)
	f.log.add("repo.Create")
	return nil
}

func (f *fakeBannerRepo) GetByID(_ context.Context, id uint) (*domain.AdBanner, error) {
	for _, b := range f.banners {
		if b.ID == id {
			return b, nil
		}
	}
	return nil, domain.NewNotFoundError("Không tìm thấy chiến dịch quảng cáo")
}

func (f *fakeBannerRepo) Update(_ context.Context, banner *domain.AdBanner) error {
	f.log.add("repo.Update")
	return nil
}

func (f *fakeBannerRepo) Delete(_ context.Context, id uint) error {
	f.log.add("repo.Delete")
	return nil
}

func (f *fakeBannerRepo) List(_ context.Context, _ domain.AdBannerFilters) ([]*domain.AdBanner, error) {
	return f.banners, nil
}

func (f *fakeBannerRepo) ListLiveAt(_ context.Context, _ time.Time) ([]*domain.AdBanner, error) {
	f.listCalls++
	return f.banners, nil // callers pre-order this slice
}

func (f *fakeBannerRepo) RecordCTAClick(_ context.Context, bannerID, employeeID uint, ctaIndex int) error {
	f.log.add("repo.RecordCTAClick")
	f.clicks = append(f.clicks, domain.AdBannerCTAClick{BannerID: bannerID, EmployeeID: employeeID, CTAIndex: ctaIndex})
	return nil
}

func (f *fakeBannerRepo) CTAClickCounts(_ context.Context, bannerIDs []uint) (map[uint]map[int]int64, error) {
	counts := make(map[uint]map[int]int64)
	for _, click := range f.clicks {
		if counts[click.BannerID] == nil {
			counts[click.BannerID] = make(map[int]int64)
		}
		counts[click.BannerID][click.CTAIndex]++
	}
	return counts, nil
}

type fakeProjectEmployeeRepo struct {
	domain.ProjectEmployeeRepository // embed; only the used method is real
	projects                         map[uint][]*domain.Project
}

func (f fakeProjectEmployeeRepo) GetActiveProjectsForEmployee(_ context.Context, employeeID uint) ([]*domain.Project, error) {
	return f.projects[employeeID], nil
}

type fakeEmployeeRepo struct {
	domain.EmployeeRepository // embed; only the used method is real
	byUserID                  map[uint]*domain.Employee
}

func (f fakeEmployeeRepo) GetByUserID(_ context.Context, userID uint) (*domain.Employee, error) {
	if emp, ok := f.byUserID[userID]; ok {
		return emp, nil
	}
	return nil, domain.NewNotFoundError("no employee")
}

type fakeCache struct {
	domain.CacheServiceUseCase // embed; only the used methods are real
	log                        *callLog
	seed                       map[string][]*domain.AdBanner
	getCalls                   int
}

func (f *fakeCache) Get(_ context.Context, key string, dest any) error {
	f.getCalls++
	if seeded, ok := f.seed[key]; ok {
		banners, ok := dest.(*[]*domain.AdBanner)
		if !ok {
			return errors.New("unexpected dest type")
		}
		*banners = seeded
		return nil
	}
	return errors.New("cache miss")
}

func (f *fakeCache) Set(_ context.Context, key string, _ any, _ time.Duration) error {
	f.log.add("cache.Set:" + key)
	return nil
}

func (f *fakeCache) DeletePattern(_ context.Context, pattern string) error {
	f.log.add("cache.DeletePattern:" + pattern)
	return nil
}

type fakeEventBus struct {
	domain.EventBus // embed
	published       int
}

func (f *fakeEventBus) Publish(_ context.Context, _ ...domain.DomainEvent) error {
	f.published++
	return nil
}

// --- helpers --------------------------------------------------------------

func testService(repo *fakeBannerRepo, peRepo fakeProjectEmployeeRepo, empRepo fakeEmployeeRepo, cache *fakeCache, _ *callLog) *Service {
	return NewService(repo, peRepo, empRepo, &fakeEventBus{}, cache, slog.New(slog.NewTextHandler(io.Discard, nil)))
}

func liveWindow() (time.Time, time.Time) {
	now := time.Now()
	return now.Add(-time.Hour), now.Add(24 * time.Hour)
}

func liveBanner(id uint, priority int, targets []uint) *domain.AdBanner {
	starts, ends := liveWindow()
	return &domain.AdBanner{
		ID: id, Title: "T" + string(rune('0'+id)), Priority: priority,
		TargetProjectIDs: targets, StartsAt: starts, EndsAt: ends, IsActive: true,
		CTAs: []domain.AdBannerCTA{{Label: "Gọi", Type: domain.AdBannerCTATypePhone, Value: "0914827988"}},
	}
}

// --- tests ----------------------------------------------------------------

func TestResolveForEmployeePicksHighestPriority(t *testing.T) {
	low := liveBanner(1, 5, []uint{12})
	high := liveBanner(2, 10, []uint{12})
	repo := &fakeBannerRepo{banners: []*domain.AdBanner{high, low}} // resolution order
	svc := testService(repo,
		fakeProjectEmployeeRepo{projects: map[uint][]*domain.Project{7: {{ID: 12}}}},
		fakeEmployeeRepo{byUserID: map[uint]*domain.Employee{100: {ID: 7}}},
		&fakeCache{log: &callLog{}}, &callLog{})

	got, err := svc.ResolveForEmployeeByUser(context.Background(), 100)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got == nil || got.ID != 2 {
		t.Fatalf("expected banner 2 (priority 10) to win, got %+v", got)
	}
}

func TestResolveForEmployeeSkipsWrongProject(t *testing.T) {
	otherProjectHigh := liveBanner(1, 10, []uint{99})
	broadcastLow := liveBanner(2, 1, nil)
	repo := &fakeBannerRepo{banners: []*domain.AdBanner{otherProjectHigh, broadcastLow}}
	svc := testService(repo,
		fakeProjectEmployeeRepo{projects: map[uint][]*domain.Project{7: {{ID: 12}}}},
		fakeEmployeeRepo{byUserID: map[uint]*domain.Employee{100: {ID: 7}}},
		&fakeCache{log: &callLog{}}, &callLog{})

	got, err := svc.ResolveForEmployeeByUser(context.Background(), 100)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got == nil || got.ID != 2 {
		t.Fatalf("expected broadcast banner to win over wrong-project banner, got %+v", got)
	}
}

func TestResolveForEmployeeNoActiveProject(t *testing.T) {
	broadcast := liveBanner(1, 1, nil)
	repo := &fakeBannerRepo{banners: []*domain.AdBanner{broadcast}}
	svc := testService(repo,
		fakeProjectEmployeeRepo{projects: map[uint][]*domain.Project{}},
		fakeEmployeeRepo{byUserID: map[uint]*domain.Employee{100: {ID: 7}}},
		&fakeCache{log: &callLog{}}, &callLog{})

	got, err := svc.ResolveForEmployeeByUser(context.Background(), 100)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != nil {
		t.Fatalf("employee with no active project must get nil, not a broadcast banner")
	}
}

func TestResolveForEmployeeNoEmployeeRecord(t *testing.T) {
	svc := testService(&fakeBannerRepo{},
		fakeProjectEmployeeRepo{projects: map[uint][]*domain.Project{}},
		fakeEmployeeRepo{byUserID: map[uint]*domain.Employee{}},
		&fakeCache{log: &callLog{}}, &callLog{})

	got, err := svc.ResolveForEmployeeByUser(context.Background(), 100)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != nil {
		t.Fatalf("user without employee record must get nil")
	}
}

func TestResolveUsesCacheAfterFirstLoad(t *testing.T) {
	repo := &fakeBannerRepo{banners: []*domain.AdBanner{liveBanner(1, 1, []uint{12})}}
	cache := &fakeCache{log: &callLog{}}
	svc := testService(repo,
		fakeProjectEmployeeRepo{projects: map[uint][]*domain.Project{7: {{ID: 12}}}},
		fakeEmployeeRepo{byUserID: map[uint]*domain.Employee{100: {ID: 7}}},
		cache, &callLog{})

	ctx := context.Background()
	if _, err := svc.ResolveForEmployeeByUser(ctx, 100); err != nil {
		t.Fatal(err)
	}
	if repo.listCalls != 1 {
		t.Fatalf("expected one repo list call, got %d", repo.listCalls)
	}
	// Seed the cache the way the real cache would have been populated.
	cache.seed = map[string][]*domain.AdBanner{liveListCacheKey: repo.banners}
	if _, err := svc.ResolveForEmployeeByUser(ctx, 100); err != nil {
		t.Fatal(err)
	}
	if repo.listCalls != 1 {
		t.Fatalf("second resolve must hit the cache, but repo was called %d times", repo.listCalls)
	}
}

func TestCreateInvalidatesAfterMutationAndPublishes(t *testing.T) {
	log := &callLog{}
	repo := &fakeBannerRepo{log: log}
	cache := &fakeCache{log: log}
	svc := testService(repo,
		fakeProjectEmployeeRepo{},
		fakeEmployeeRepo{},
		cache, log)

	starts, _ := liveWindow()
	banner := liveBanner(0, 0, []uint{12})
	banner.StartsAt = starts
	if _, err := svc.Create(context.Background(), banner); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !log.hasAfter("cache.DeletePattern:"+cacheInvalidationPattern, "repo.Create") {
		t.Fatalf("cache must be invalidated AFTER the create committed; log: %v", log.events)
	}
}

func TestCreateRejectsInvalidBanner(t *testing.T) {
	svc := testService(&fakeBannerRepo{log: &callLog{}},
		fakeProjectEmployeeRepo{}, fakeEmployeeRepo{},
		&fakeCache{log: &callLog{}}, &callLog{})

	banner := liveBanner(0, 0, []uint{12})
	banner.CTAs = nil // invalid: at least one CTA required
	if _, err := svc.Create(context.Background(), banner); err == nil {
		t.Fatal("expected validation error for banner without CTAs")
	}
}

func TestRecordCTAClickSkipsDeadBanner(t *testing.T) {
	log := &callLog{}
	repo := &fakeBannerRepo{log: log}
	dead := liveBanner(5, 1, nil)
	dead.EndsAt = time.Now().Add(-time.Minute)
	repo.banners = []*domain.AdBanner{dead}
	svc := testService(repo,
		fakeProjectEmployeeRepo{},
		fakeEmployeeRepo{byUserID: map[uint]*domain.Employee{100: {ID: 7}}},
		&fakeCache{log: log}, log)

	if err := svc.RecordCTAClick(context.Background(), 5, 100, 0); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(repo.clicks) != 0 {
		t.Fatalf("dead banner must not record clicks, got %d", len(repo.clicks))
	}
}

func TestRecordCTAClickUsesEmployeeIdentity(t *testing.T) {
	log := &callLog{}
	repo := &fakeBannerRepo{log: log}
	repo.banners = []*domain.AdBanner{liveBanner(5, 1, nil)}
	svc := testService(repo,
		fakeProjectEmployeeRepo{},
		fakeEmployeeRepo{byUserID: map[uint]*domain.Employee{100: {ID: 7}}},
		&fakeCache{log: log}, log)

	if err := svc.RecordCTAClick(context.Background(), 5, 100, 1); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(repo.clicks) != 1 || repo.clicks[0].EmployeeID != 7 || repo.clicks[0].CTAIndex != 1 {
		t.Fatalf("click must record the employee id (7), got %+v", repo.clicks)
	}
}

func TestListWithStatsAggregatesClicks(t *testing.T) {
	repo := &fakeBannerRepo{}
	repo.banners = []*domain.AdBanner{liveBanner(5, 1, nil)}
	repo.clicks = []domain.AdBannerCTAClick{
		{BannerID: 5, EmployeeID: 1, CTAIndex: 0},
		{BannerID: 5, EmployeeID: 2, CTAIndex: 0},
		{BannerID: 5, EmployeeID: 1, CTAIndex: 1},
	}
	svc := testService(repo, fakeProjectEmployeeRepo{}, fakeEmployeeRepo{},
		&fakeCache{log: &callLog{}}, &callLog{})

	_, counts, err := svc.ListWithStats(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if counts[5][0] != 2 || counts[5][1] != 1 {
		t.Fatalf("unexpected counts: %+v", counts)
	}
}
