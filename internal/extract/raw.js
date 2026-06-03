// Diagnostic raw dumper (the --raw flag). Runs in the page's MAIN world and
// returns the UNTOUCHED client/buyer object(s) Upwork ships, so we can see every
// field before deciding what to normalize. Mirrors extract.js's page detection
// and Nuxt3 devalue decode, but maps nothing — it just dumps raw shapes.
() => {
  const nuxt = window.__NUXT__ || {};
  const state = nuxt.state || {};

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
      if (state[k] && Array.isArray(state[k].jobs) && state[k].jobs.length) return { key: k, jobs: state[k].jobs };
    }
    for (const k in state) {
      const v = state[k];
      if (v && Array.isArray(v.jobs) && v.jobs.length && v.jobs[0] && 'ciphertext' in v.jobs[0]) return { key: k, jobs: v.jobs };
    }
    return null;
  }

  // devalue unflatten (Nuxt3 __NUXT_DATA__) — same as extract.js
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
          else hydrated[index] = hydrate(value[1]);
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

  // Drop bulky/noisy keys so the dump stays readable; keep everything else raw.
  function pruneJob(j) {
    const out = {};
    for (const k in j) {
      if (k === 'description' || k === 'skills' || k === 'attrs' || k === 'questions') continue;
      out[k] = j[k];
    }
    return out;
  }

  const out = { pageType: type, source: null, jobKeys: [], samples: [] };

  if (type === 'feed' || type === 'search') {
    const found = findJobsArray();
    if (found) {
      out.source = found.key;
      const jobs = found.jobs;
      if (jobs[0]) out.jobKeys = Object.keys(jobs[0]);
      for (let i = 0; i < Math.min(3, jobs.length); i++) {
        const j = jobs[i];
        const client = j.client || j.buyer || {};
        out.samples.push({
          title: j.title,
          ciphertext: j.ciphertext,
          clientKeys: Object.keys(client),
          client: client,            // RAW, untouched
          jobScalar: pruneJob(j),    // remaining raw job fields (no desc/skills)
        });
      }
    }
  } else if (type === 'job') {
    let buyer = null, job = null;
    const root = Object.keys(state).length ? state : null;
    if (root) {
      job = bfsFind(root, (n) => 'ciphertext' in n && 'title' in n);
      buyer = bfsFind(root, (n) => (n.totalReviews !== undefined || n.totalSpent !== undefined || n.stats !== undefined));
    }
    if (!buyer || !job) {
      const el = document.getElementById('__NUXT_DATA__');
      if (el) {
        try {
          const data = unflatten(JSON.parse(el.textContent));
          if (!job) job = bfsFind(data, (n) => 'ciphertext' in n && 'title' in n);
          if (!buyer) buyer = bfsFind(data, (n) =>
            (n.totalReviews !== undefined || n.totalSpent !== undefined || n.stats !== undefined) &&
            (n.location !== undefined || n.totalFeedback !== undefined || n.totalHires !== undefined || n.stats !== undefined));
        } catch (e) { out.error = String(e); }
      }
    }
    out.source = '__NUXT_DATA__/state';
    if (job) out.jobKeys = Object.keys(job);
    const client = buyer || (job && (job.client || job.buyer)) || {};
    out.samples.push({
      title: job && job.title,
      ciphertext: job && job.ciphertext,
      clientKeys: Object.keys(client),
      client: client,                       // RAW, untouched
      clientStats: client && client.stats,  // surfaced explicitly if present
    });
  }

  return JSON.stringify(out, null, 2);
}
