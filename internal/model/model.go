// Package model defines the normalized data shapes the tool exports.
// The browser-side extractor (internal/extract/extract.js) produces JSON that
// unmarshals directly into Result, so the JSON tags here are the contract
// between the in-page JavaScript and Go.
package model

// PageType classifies the Upwork page currently loaded in the browser.
type PageType string

const (
	PageFeed    PageType = "feed"
	PageSearch  PageType = "search"
	PageJob     PageType = "job"
	PageAll     PageType = "all" // several feeds merged + deduplicated
	PageUnknown PageType = "unknown"
)

// Client is the normalized client/buyer profile attached to a job. The feeds use
// two payload shapes: a "lean" one (most-recent, best-matches) and a "rich" one
// (my-feed) that also carries totalPostedJobs and companyOrgUid; fields absent on
// a given feed are left at their zero value. Note: AvgHourlyRate and TotalHours
// are NOT in any feed payload — they exist only on the single-job client panel,
// which sits behind Cloudflare and can't be loaded by the automated browser.
type Client struct {
	PaymentVerified  bool    `json:"paymentVerified" xml:"paymentVerified"`
	TotalSpent       float64 `json:"totalSpent" xml:"totalSpent"`
	TotalReviews     int     `json:"totalReviews" xml:"totalReviews"`
	Rating           float64 `json:"rating" xml:"rating"` // totalFeedback, 0..5
	TotalHires       int     `json:"totalHires" xml:"totalHires"`
	TotalPostedJobs  int     `json:"totalPostedJobs" xml:"totalPostedJobs"` // my-feed only
	Country          string  `json:"country" xml:"country"`
	City             string  `json:"city" xml:"city"`
	TopClient        bool    `json:"topClient" xml:"topClient"`
	FinancialPrivacy bool    `json:"financialPrivacy" xml:"financialPrivacy"`
	// LastRecruitingActivity is when the client last acted on a posting (ISO
	// 8601); empty when none. A freshness/responsiveness signal from the feed.
	LastRecruitingActivity string `json:"lastRecruitingActivity" xml:"lastRecruitingActivity"`
	// CompanyOrgUID is a stable opaque ID for the client's org (my-feed only),
	// usable to recognize the same client across postings.
	CompanyOrgUID string `json:"companyOrgUid" xml:"companyOrgUid"`
}

// Job is the normalized job record. Fields absent on a given page type are
// left at their zero value.
type Job struct {
	ID                string  `json:"id" xml:"id"` // ciphertext, e.g. ~021234...
	UID               string  `json:"uid" xml:"uid"`
	Recno             string  `json:"recno" xml:"recno"` // Upwork record number
	URL               string  `json:"url" xml:"url"`
	Title             string  `json:"title" xml:"title"`
	Description       string  `json:"description" xml:"description"`
	Type              string  `json:"type" xml:"type"` // hourly | fixed
	HourlyMin         float64 `json:"hourlyMin" xml:"hourlyMin"`
	HourlyMax         float64 `json:"hourlyMax" xml:"hourlyMax"`
	FixedBudget       float64 `json:"fixedBudget" xml:"fixedBudget"`
	WeeklyBudget      float64 `json:"weeklyBudget" xml:"weeklyBudget"`
	Engagement        string  `json:"engagement" xml:"engagement"`
	Duration          string  `json:"duration" xml:"duration"`
	ExperienceLevel   string  `json:"experienceLevel" xml:"experienceLevel"`
	FreelancersToHire int     `json:"freelancersToHire" xml:"freelancersToHire"`
	ProposalsTier     string  `json:"proposalsTier" xml:"proposalsTier"`
	TotalApplicants   int     `json:"totalApplicants" xml:"totalApplicants"` // exact count (my-feed)
	Premium           bool    `json:"premium" xml:"premium"`
	Applied           bool    `json:"applied" xml:"applied"`
	Enterprise        bool    `json:"enterprise" xml:"enterprise"`
	JobStatus         string  `json:"jobStatus" xml:"jobStatus"` // e.g. Open (my-feed)
	IsLocal           bool    `json:"isLocal" xml:"isLocal"`
	// PrefFreelancerLocation lists countries the client prefers (my-feed);
	// PrefFreelancerLocationMandatory says whether that preference is enforced.
	PrefFreelancerLocation          []string `json:"prefFreelancerLocation" xml:"prefFreelancerLocation>location"`
	PrefFreelancerLocationMandatory bool     `json:"prefFreelancerLocationMandatory" xml:"prefFreelancerLocationMandatory"`
	CreatedOn                       string   `json:"createdOn" xml:"createdOn"`
	PublishedOn                     string   `json:"publishedOn" xml:"publishedOn"`
	RenewedOn                       string   `json:"renewedOn" xml:"renewedOn"`
	ConnectPrice                    int      `json:"connectPrice" xml:"connectPrice"`
	Position                        int      `json:"position" xml:"position"` // rank within the feed
	Skills                          []string `json:"skills" xml:"skills>skill"`
	Tags                            []string `json:"tags" xml:"tags>tag"` // annotations.tags, e.g. firstJobPost
	Client                          Client   `json:"client" xml:"client"`
}

// Result is the top-level extraction output for one page.
type Result struct {
	PageType PageType `json:"pageType" xml:"pageType,attr"`
	Count    int      `json:"count" xml:"count,attr"`
	Jobs     []Job    `json:"jobs" xml:"jobs>job"`
	Job      *Job     `json:"job,omitempty" xml:"-"`
	Error    string   `json:"error,omitempty" xml:"-"`
}

// Exportable reports whether the page yielded anything worth writing.
func (r *Result) Exportable() bool {
	return r != nil && (len(r.Jobs) > 0 || r.Job != nil)
}
