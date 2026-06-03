// In-page extractor. Runs in the page's MAIN world so it can see window.__NUXT__.
// Returns a JSON string matching Go's model.Result. Handles three page shapes:
//   - feed   (Nuxt2): jobs at __NUXT__.state.<feed*>.jobs
//   - search (Nuxt2): jobs at __NUXT__.state.jobsSearch.jobs
//   - job    (Nuxt3): devalue payload in <script id="__NUXT_DATA__"> (decoded here)
// We never regex the raw HTML; the browser materializes the data for us.
() => {
  const UPWORK = 'https://www.upwork.com';
  const jobURL = (c) => (c ? UPWORK + '/jobs/' + c : '');
  // Upwork mixes numbers and numeric strings, plus several money-object shapes
  // across feeds: {amount} (most-recent), {rawValue, currency, displayValue}
  // (my-feed). Coerce everything numeric explicitly, trying each known wrapper.
  const toNum = (v) => {
    if (v == null) return 0;
    if (typeof v === 'object') {
      if (v.amount != null) return toNum(v.amount);
      if (v.rawValue != null) return toNum(v.rawValue);
      if (v.displayValue != null) return toNum(v.displayValue);
      return 0;
    }
    const n = Number(v);
    return isFinite(n) ? n : 0;
  };
  const toInt = (v) => Math.trunc(toNum(v));

  function normClient(c) {
    c = c || {};
    const loc = c.location || {};
    return {
      paymentVerified: !!(c.isPaymentVerified || c.paymentVerificationStatus === 'VERIFIED' ||
        c.paymentVerificationStatus === 1 || c.verificationStatus === 'VERIFIED'),
      totalSpent: toNum(c.totalSpent),
      totalReviews: toInt(c.totalReviews),
      rating: toNum(c.totalFeedback != null ? c.totalFeedback : c.score),
      totalHires: toInt(c.totalHires),
      totalPostedJobs: toInt(c.totalPostedJobs),
      country: loc.country || c.country || '',
      city: loc.city || '',
      topClient: !!c.topClient,
      financialPrivacy: !!c.hasFinancialPrivacy,
      lastRecruitingActivity: c.lastRecruitingActivity || '',
      companyOrgUid: String(c.companyOrgUid || ''),
    };
  }

  function budgetType(j) {
    const hb = j.hourlyBudget;
    if (hb && (hb.min || hb.max || hb.type)) return 'hourly';
    if (toNum(j.amount) > 0) return 'fixed';
    if (j.type === 1 || j.type === '1') return 'hourly';
    if (j.type === 2 || j.type === '2') return 'fixed';
    return String(j.type || '');
  }

  // Experience level differs by feed: lean feeds carry tierText ("Expert");
  // my-feed carries contractorTier ("EXPERT"/"ENTRY_LEVEL"). Prefer tierText;
  // fall back to a title-cased contractorTier so the column stays consistent.
  // (tierLabel is deliberately ignored — it's the UI string "Experience Level".)
  function experienceLevel(j) {
    if (j.tierText) return j.tierText;
    const t = j.contractorTier;
    if (!t) return '';
    return String(t).toLowerCase().replace(/_/g, ' ').replace(/\b\w/g, (m) => m.toUpperCase());
  }

  function normJob(j) {
    j = j || {};
    const hb = j.hourlyBudget || {};
    const skills = [];
    (j.skills || []).forEach((s) => { const n = s.prefLabel || s.prettyName || s.name; if (n) skills.push(n); });
    (j.attrs || []).forEach((s) => { const n = s.prefLabel || s.prettyName; if (n && skills.indexOf(n) < 0) skills.push(n); });
    const tags = [];
    const annTags = (j.annotations && j.annotations.tags) || [];
    if (Array.isArray(annTags)) annTags.forEach((t) => { if (t) tags.push(String(t)); });
    const prefLoc = Array.isArray(j.prefFreelancerLocation) ? j.prefFreelancerLocation.map(String) : [];
    const client = j.client || j.buyer || {};
    return {
      id: j.ciphertext || j.uid || '',
      uid: String(j.uid || ''),
      recno: String(j.recno || ''),
      url: jobURL(j.ciphertext),
      title: j.title || '',
      description: j.description || '',
      type: budgetType(j),
      hourlyMin: toNum(hb.min),
      hourlyMax: toNum(hb.max),
      fixedBudget: toNum(j.amount),
      weeklyBudget: toNum(j.weeklyBudget),
      engagement: j.engagement || '',
      duration: j.durationLabel || j.duration || '',
      experienceLevel: experienceLevel(j),
      freelancersToHire: toInt(j.freelancersToHire),
      proposalsTier: j.proposalsTier || '',
      totalApplicants: toInt(j.totalApplicants),
      premium: !!j.premium,
      applied: !!j.isApplied,
      enterprise: !!j.enterpriseJob,
      jobStatus: j.jobStatus || '',
      isLocal: !!j.isLocal,
      prefFreelancerLocation: prefLoc,
      prefFreelancerLocationMandatory: !!j.prefFreelancerLocationMandatory,
      createdOn: j.createdOn || '',
      publishedOn: j.publishedOn || '',
      renewedOn: j.renewedOn || '',
      connectPrice: toInt(j.connectPrice),
      position: toInt(j.position),
      skills: skills,
      tags: tags,
      client: normClient(client),
    };
  }

  const nuxt = window.__NUXT__ || {};
  const state = nuxt.state || {};

  // --- page type: URL first (authoritative on live Upwork), then a payload
  // probe (resilient to client-side nav and works for saved file:// samples) ---
  function typeFromURL() {
    const path = location.pathname;
    if (/\/nx\/search\/jobs|\/search\//.test(path)) return 'search';
    if (/\/nx\/find-work|\/ab\/find-work|\/feed\b/.test(path)) return 'feed';
    if (/\/jobs\/~|\/nx\/job-details|\/jobs\//.test(path)) return 'job';
    return 'unknown';
  }
  function typeFromContent() {
    if (state.jobsSearch && Array.isArray(state.jobsSearch.jobs)) return 'search';
    for (const k in state) {
      if (state[k] && Array.isArray(state[k].jobs) && state[k].jobs.length) return 'feed';
    }
    if (document.getElementById('__NUXT_DATA__')) return 'job';
    return 'unknown';
  }
  let type = typeFromURL();
  if (type === 'unknown') type = typeFromContent();

  function findJobsArray() {
    const prefer = ['jobsSearch', 'feedBestMatch', 'feedMostRecent', 'myFeed', 'feed'];
    for (const k of prefer) {
      if (state[k] && Array.isArray(state[k].jobs) && state[k].jobs.length) return state[k].jobs;
    }
    for (const k in state) {
      const v = state[k];
      if (v && Array.isArray(v.jobs) && v.jobs.length && v.jobs[0] && 'ciphertext' in v.jobs[0]) return v.jobs;
    }
    return null;
  }

  // --- devalue unflatten (Nuxt3 __NUXT_DATA__) ---
  function unflatten(parsed) {
    if (typeof parsed === 'number') return parsed;
    if (!Array.isArray(parsed) || !parsed.length) return null;
    const values = parsed;
    const hydrated = new Array(values.length);
    const UNDEF = -1, HOLE = -2, NAN = -3, POSINF = -4, NEGINF = -5, NEGZERO = -6;
    function hydrate(index) {
      if (index === UNDEF) return undefined;
      if (index === NAN) return NaN;
      if (index === POSINF) return Infinity;
      if (index === NEGINF) return -Infinity;
      if (index === NEGZERO) return -0;
      if (index in hydrated) return hydrated[index];
      const value = values[index];
      if (!value || typeof value !== 'object') {
        hydrated[index] = value;
      } else if (Array.isArray(value)) {
        if (typeof value[0] === 'string') {
          const t = value[0];
          if (t === 'Date') hydrated[index] = new Date(value[1]);
          else if (t === 'Set') { const s = new Set(); hydrated[index] = s; for (let i = 1; i < value.length; i++) s.add(hydrate(value[i])); }
          else if (t === 'Map') { const m = new Map(); hydrated[index] = m; for (let i = 1; i < value.length; i += 2) m.set(hydrate(value[i]), hydrate(value[i + 1])); }
          else hydrated[index] = hydrate(value[1]); // Reactive/Ref/ShallowRef/NuxtError/... -> unwrap
        } else {
          const arr = []; hydrated[index] = arr;
          for (const n of value) arr.push(n === HOLE ? undefined : hydrate(n));
        }
      } else {
        const obj = {}; hydrated[index] = obj;
        for (const key in value) obj[key] = hydrate(value[key]);
      }
      return hydrated[index];
    }
    return hydrate(0);
  }

  function bfsFind(root, pred) {
    const seen = new Set();
    const q = [root];
    while (q.length) {
      const n = q.shift();
      if (!n || typeof n !== 'object' || seen.has(n)) continue;
      seen.add(n);
      if (!Array.isArray(n) && pred(n)) return n;
      if (Array.isArray(n)) { for (const x of n) q.push(x); }
      else { for (const k in n) q.push(n[k]); }
    }
    return null;
  }

  const result = { pageType: type, count: 0, jobs: [], job: null };

  if (type === 'feed' || type === 'search') {
    const arr = findJobsArray();
    if (arr) { result.jobs = arr.map(normJob); result.count = result.jobs.length; }
  } else if (type === 'job') {
    let job = null;
    const root = Object.keys(state).length ? state : null;
    if (root) job = bfsFind(root, (n) => 'ciphertext' in n && 'title' in n && ('description' in n || 'engagement' in n));
    let client = null;
    if (!job) {
      const el = document.getElementById('__NUXT_DATA__');
      if (el) {
        try {
          const data = unflatten(JSON.parse(el.textContent));
          job = bfsFind(data, (n) => 'ciphertext' in n && 'title' in n && ('description' in n || 'engagement' in n));
          client = bfsFind(data, (n) => (n.totalReviews !== undefined || n.totalSpent !== undefined) &&
            (n.location !== undefined || n.totalFeedback !== undefined || n.totalHires !== undefined));
        } catch (e) { result.error = String(e); }
      }
    }
    if (job) {
      const nj = normJob(job);
      const c = client || job.client || job.buyer;
      if (c) nj.client = normClient(c);
      // STEP 6 TODO: the single-job buyer uses a different schema than feed/search
      // (buyer.stats.{score, feedbackCount, totalAssignments}, buyer.location,
      // buyer.company) with no totalSpent/totalReviews — add a dedicated mapper so
      // client rating/spend/hires populate for single-job exports too.
      result.job = nj; result.jobs = [nj]; result.count = 1;
    }
  }
  return JSON.stringify(result);
}
