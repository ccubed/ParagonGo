// Standard Picker configurations for common admin entity types.
//
// Each entry is a plain object that can be spread into Picker.open():
//
//   Picker.open({
//     ...PickerConfigs.buffs,
//     onSelect: (buff) => appendBuffChip(listEl, buff.BuffId, buff.Name),
//   });
//
//   Picker.open({
//     ...PickerConfigs.buffs,
//     title:    'Select Worn Buff',
//     filter:   b => !b.Secret,
//     multi:    true,
//     selected: collectBuffIds('wornbuffids-list'),
//     onSelect: (buffs) => replaceBuffChips('wornbuffids-list', buffs),
//   });

const PickerConfigs = {

    buffs: {
        title:   'Select Buff',
        idKey:   'BuffId',
        columns: [
            { key: 'BuffId',      label: '#',    width: '4rem',  mono: true },
            {
                key: 'Name', label: 'Name', flex: true,
                // Render the name with the description as a muted subtitle so the
                // picker conveys what each buff actually does.
                render: (name, buff) => {
                    const wrap = document.createElement('div');
                    const n = document.createElement('div');
                    n.style.fontWeight = '600';
                    n.textContent = name == null ? '' : String(name);
                    wrap.appendChild(n);
                    if (buff.Description) {
                        const d = document.createElement('div');
                        d.style.cssText = 'font-size:0.78rem;color:var(--color-text-faint);margin-top:0.1rem;white-space:normal;';
                        d.textContent = buff.Description;
                        wrap.appendChild(d);
                    }
                    return wrap;
                },
            },
            { key: 'TriggerRate', label: 'Rate', width: '6rem' },
        ],
        searchKeys: ['BuffId', 'Name', 'Description'],
        sort: (a, b) => a.BuffId - b.BuffId,
        // buffs API returns { data: { specs: { ... } } } - unwrap via source fn
        source: async () => {
            const res = await AdminAPI.get('/admin/api/v1/buffs');
            if (!res.ok) throw new Error(res.error || 'Failed to load buffs');
            const specs = (res.data && res.data.data && res.data.data.specs) || {};
            return Object.values(specs).sort((a, b) => a.BuffId - b.BuffId);
        },
    },

    items: {
        title:   'Select Item',
        source:  '/admin/api/v1/items',
        idKey:   'ItemId',
        columns: [
            { key: 'ItemId', label: '#',    width: '4rem', mono: true },
            { key: 'Name',   label: 'Name', flex: true },
            { key: 'Type',   label: 'Type', width: '8rem' },
        ],
        searchKeys: ['ItemId', 'Name'],
        sort: (a, b) => a.ItemId - b.ItemId,
    },

    quests: {
        title:   'Select Quest',
        source:  '/admin/api/v1/quests',
        idKey:   'QuestId',
        columns: [
            { key: 'QuestId', label: '#',    width: '4rem', mono: true },
            { key: 'Name',    label: 'Name', flex: true },
        ],
        searchKeys: ['QuestId', 'Name'],
        sort: (a, b) => a.QuestId - b.QuestId,
    },

    mobs: {
        title:   'Select Mob',
        idKey:   'MobId',
        columns: [
            { key: 'MobId', label: '#',    width: '4rem', mono: true },
            { key: '_name', label: 'Name', flex: true },
            { key: 'Zone',  label: 'Zone', width: '10rem' },
        ],
        searchKeys: ['MobId', '_name', 'Zone'],
        sort: (a, b) => a.MobId - b.MobId,
        source: async () => {
            const res = await AdminAPI.get('/admin/api/v1/mobs');
            if (!res.ok) throw new Error(res.error || 'Failed to load mobs');
            const items = (res.data && res.data.data) || [];
            return items.map(m => ({
                ...m,
                _name: (m.Character && m.Character.Name) || '(unnamed)',
            }));
        },
    },

    spells: {
        title:      'Select Spell',
        source:     '/admin/api/v2/spells',
        idKey:      'SpellId',
        columns: [
            { key: 'SpellId', label: 'ID',     width: '10rem', mono: true },
            { key: 'Name',    label: 'Name',   flex: true },
            { key: 'Type',    label: 'Type',   width: '8rem' },
            { key: 'School',  label: 'School', width: '8rem' },
        ],
        searchKeys: ['SpellId', 'Name'],
        sort: (a, b) => a.Name.localeCompare(b.Name),
    },

    mutators: {
        title:      'Select Mutator',
        source:     '/admin/api/v1/mutators',
        idKey:      'MutatorId',
        columns: [
            { key: 'MutatorId',  label: 'ID',         flex: true, mono: true },
            { key: 'DecayRate',  label: 'Decay Rate',  width: '8rem' },
            { key: 'RespawnRate', label: 'Respawn',    width: '8rem' },
        ],
        searchKeys: ['MutatorId'],
        sort: (a, b) => a.MutatorId.localeCompare(b.MutatorId),
    },

};

// buffName(id) - synchronously resolves a buff name from the AdminAPI cache.
// Returns '#ID Name' if the cache is warm (i.e. the buffs API has already been
// fetched this session), or '#ID' if it hasn't been fetched yet.
PickerConfigs.buffName = function (id) {
    const cached = AdminAPI._buffNameCache || (AdminAPI._buffNameCache = {});
    if (cached[id] !== undefined) return cached[id];
    // Try to read from the already-cached AdminAPI GET response (synchronous).
    const entry = (window._adminBuffSpecs || {})[String(id)];
    if (entry) {
        cached[id] = '#' + id + ' ' + entry.Name;
        return cached[id];
    }
    return '#' + id;
};

// buffDescription(id) - synchronously resolves a buff's description from the
// AdminAPI cache. Returns '' if the cache isn't warm yet or the buff has none.
PickerConfigs.buffDescription = function (id) {
    const entry = (window._adminBuffSpecs || {})[String(id)];
    return (entry && entry.Description) || '';
};

// Pre-warm _adminBuffSpecs whenever buffs are fetched so buffName() works
// synchronously on subsequent calls.
AdminAPI.get('/admin/api/v1/buffs').then(res => {
    if (res.ok && res.data && res.data.data && res.data.data.specs) {
        window._adminBuffSpecs = res.data.data.specs;
    }
});

// Pre-warm _adminQuestSpecs (keyed by QuestId) so quest tooltips can resolve a
// token's quest name and step description synchronously at hover time.
AdminAPI.get('/admin/api/v1/quests').then(res => {
    if (res.ok && res.data && Array.isArray(res.data.data)) {
        const byId = {};
        for (const q of res.data.data) byId[String(q.QuestId)] = q;
        window._adminQuestSpecs = byId;
    }
});

// attachHoverTooltip - registers a hover provider for a CSS `selector`. When the
// pointer is over a matching element, `buildContent(target)` is called to build a
// DOM node shown in a single shared, themed floating tooltip (returning null
// suppresses it). Content is built lazily on hover, so it works even when the
// underlying spec caches warm up after the chips are rendered. A single set of
// document listeners and one tooltip element are shared across all providers, so
// providers for different selectors never fight over the tooltip.
const attachHoverTooltip = (function () {
    'use strict';

    const providers = []; // { selector, build }
    let tipEl = null;
    let bound = false;

    function ensureTip() {
        if (tipEl) return tipEl;
        tipEl = document.createElement('div');
        tipEl.className = 'hover-tooltip';
        tipEl.style.cssText = [
            'position:fixed', 'z-index:10000', 'pointer-events:none',
            'max-width:320px', 'padding:0.5rem 0.65rem', 'border-radius:6px',
            'background:var(--color-surface-raised)', 'color:var(--color-text)',
            'border:1px solid var(--color-border-medium)',
            'box-shadow:0 4px 16px var(--color-shadow)',
            'font-size:0.8rem', 'line-height:1.35', 'display:none',
        ].join(';');
        document.body.appendChild(tipEl);
        return tipEl;
    }

    function position(e) {
        if (!tipEl || tipEl.style.display === 'none') return;
        const pad = 14;
        const rect = tipEl.getBoundingClientRect();
        let x = e.clientX + pad;
        let y = e.clientY + pad;
        if (x + rect.width > window.innerWidth - 8) x = e.clientX - rect.width - pad;
        if (y + rect.height > window.innerHeight - 8) y = e.clientY - rect.height - pad;
        tipEl.style.left = Math.max(8, x) + 'px';
        tipEl.style.top = Math.max(8, y) + 'px';
    }

    function hide() { if (tipEl) tipEl.style.display = 'none'; }

    // Return the first provider whose selector matches an ancestor of `node`,
    // along with the matched element.
    function match(node) {
        if (!node || !node.closest) return null;
        for (const p of providers) {
            const el = node.closest(p.selector);
            if (el) return { el: el, build: p.build };
        }
        return null;
    }

    function bind() {
        if (bound) return;
        bound = true;

        document.addEventListener('mouseover', function (e) {
            const m = match(e.target);
            if (!m) return;
            const content = m.build(m.el);
            if (!content) { hide(); return; }
            const tip = ensureTip();
            tip.innerHTML = '';
            tip.appendChild(content);
            tip.style.display = 'block';
            position(e);
        });
        document.addEventListener('mousemove', function (e) {
            if (!tipEl || tipEl.style.display === 'none') return;
            if (!match(e.target)) { hide(); return; }
            position(e);
        });
        document.addEventListener('mouseout', function (e) {
            if (!match(e.target)) return;
            // Keep the tooltip if we're still moving within a matching element.
            if (match(e.relatedTarget)) return;
            hide();
        });
    }

    return function attach(selector, buildContent) {
        providers.push({ selector: selector, build: buildContent });
        bind();
    };
})();

// BuffTooltip - any element carrying a `data-buff-id` attribute (e.g. selected
// buff chips on items/mobs/pets/races/rooms/mutators pages) gets a hover tooltip
// showing the buff name and description.
attachHoverTooltip('[data-buff-id]', function (target) {
    const id = target.getAttribute('data-buff-id');
    if (id == null || id === '') return null;
    const spec = (window._adminBuffSpecs || {})[String(id)];
    if (!spec) return null; // cache not warm / unknown buff - skip silently

    const wrap = document.createElement('div');
    const title = document.createElement('div');
    title.style.cssText = 'font-weight:700;margin-bottom:0.15rem;';
    title.textContent = '#' + id + ' ' + (spec.Name || '');
    wrap.appendChild(title);

    const body = document.createElement('div');
    if (spec.Description) {
        body.style.cssText = 'color:var(--color-text-muted);';
        body.textContent = spec.Description;
    } else {
        body.style.cssText = 'color:var(--color-text-faint);font-style:italic;';
        body.textContent = 'No description.';
    }
    wrap.appendChild(body);
    return wrap;
});

// QuestTooltip - any element carrying a `data-quest-token` attribute (e.g. quest
// flag chips on mobs/rooms/mapper pages) gets a hover tooltip showing the quest
// name and the matching step's description. The token format is `questId-stepId`.
attachHoverTooltip('[data-quest-token]', function (target) {
    const token = target.getAttribute('data-quest-token');
    if (token == null || token === '') return null;
    const dash = token.indexOf('-');
    const questId = dash >= 0 ? token.slice(0, dash) : token;
    const stepId  = dash >= 0 ? token.slice(dash + 1) : '';
    const quest = (window._adminQuestSpecs || {})[String(questId)];
    if (!quest) return null; // cache not warm / unknown quest - skip silently

    const wrap = document.createElement('div');
    const title = document.createElement('div');
    title.style.cssText = 'font-weight:700;margin-bottom:0.15rem;';
    title.textContent = '#' + questId + ' ' + (quest.Name || '');
    wrap.appendChild(title);

    let step = null;
    if (stepId && Array.isArray(quest.Steps)) {
        step = quest.Steps.find(s => String(s.Id) === stepId) || null;
    }

    const body = document.createElement('div');
    if (step) {
        const label = document.createElement('span');
        label.style.cssText = 'font-weight:600;';
        label.textContent = stepId + ': ';
        body.appendChild(label);
        const desc = document.createElement('span');
        desc.style.cssText = 'color:var(--color-text-muted);';
        desc.textContent = step.Description || '(no description)';
        body.appendChild(desc);
    } else if (stepId) {
        body.style.cssText = 'color:var(--color-text-faint);font-style:italic;';
        body.textContent = 'Step "' + stepId + '" not found.';
    } else {
        body.style.cssText = 'color:var(--color-text-faint);font-style:italic;';
        body.textContent = quest.Description || 'No description.';
    }
    wrap.appendChild(body);
    return wrap;
});

// Pre-warm _adminItemSpecs (keyed by ItemId) so item info-card tooltips can
// resolve an item's details synchronously at hover time.
AdminAPI.get('/admin/api/v1/items').then(res => {
    if (res.ok && res.data && Array.isArray(res.data.data)) {
        const byId = {};
        for (const it of res.data.data) byId[String(it.ItemId)] = it;
        window._adminItemSpecs = byId;
    }
});

// ItemTooltip - any element carrying a `data-item-id` attribute (e.g. selected
// item chips / spawn displays on mobs/users/rooms pages) gets a hover "info
// card" with at-a-glance details: name, type/subtype, description, quest token.
attachHoverTooltip('[data-item-id]', function (target) {
    const id = target.getAttribute('data-item-id');
    if (id == null || id === '' || id === '0') return null;
    const spec = (window._adminItemSpecs || {})[String(id)];
    if (!spec) return null; // cache not warm / unknown item - skip silently

    const wrap = document.createElement('div');

    const title = document.createElement('div');
    title.style.cssText = 'font-weight:700;margin-bottom:0.15rem;';
    title.textContent = '#' + id + ' ' + (spec.Name || '');
    wrap.appendChild(title);

    const type = (spec.Type || '').toString();
    const subtype = (spec.Subtype || '').toString();
    if (type || subtype) {
        const t = document.createElement('div');
        t.style.cssText = 'color:var(--color-text-subtle);font-size:0.74rem;text-transform:uppercase;letter-spacing:0.03em;margin-bottom:0.2rem;';
        t.textContent = subtype ? (type + ' / ' + subtype) : type;
        wrap.appendChild(t);
    }

    const desc = document.createElement('div');
    if (spec.Description) {
        desc.style.cssText = 'color:var(--color-text-muted);';
        desc.textContent = spec.Description;
    } else {
        desc.style.cssText = 'color:var(--color-text-faint);font-style:italic;';
        desc.textContent = 'No description.';
    }
    wrap.appendChild(desc);

    if (spec.QuestToken) {
        const q = document.createElement('div');
        q.style.cssText = 'margin-top:0.3rem;font-size:0.75rem;color:var(--color-text-subtle);';
        const label = document.createElement('span');
        label.style.fontWeight = '600';
        label.textContent = 'Quest token: ';
        q.appendChild(label);
        const tok = document.createElement('span');
        tok.style.fontFamily = 'monospace';
        tok.textContent = spec.QuestToken;
        q.appendChild(tok);
        wrap.appendChild(q);
    }

    return wrap;
});

// Pre-warm _adminSpellSpecs (keyed by SpellId) so spell tooltips can resolve a
// spell's details synchronously at hover time.
AdminAPI.get('/admin/api/v2/spells').then(res => {
    if (res.ok && res.data && Array.isArray(res.data.data)) {
        const byId = {};
        for (const sp of res.data.data) byId[String(sp.SpellId)] = sp;
        window._adminSpellSpecs = byId;
    }
});

// SpellTooltip - any element carrying a `data-spell-id` attribute (e.g. spellbook
// chips on mobs/users pages, skill-training rows on rooms) gets a hover tooltip
// showing the spell name, type, and description.
attachHoverTooltip('[data-spell-id]', function (target) {
    const id = target.getAttribute('data-spell-id');
    if (id == null || id === '') return null;
    const spec = (window._adminSpellSpecs || {})[String(id)];
    if (!spec) return null; // cache not warm / unknown spell - skip silently

    const wrap = document.createElement('div');

    const title = document.createElement('div');
    title.style.cssText = 'font-weight:700;margin-bottom:0.15rem;';
    title.textContent = spec.Name || id;
    wrap.appendChild(title);

    const type = (spec.Type || '').toString();
    if (type) {
        const t = document.createElement('div');
        t.style.cssText = 'color:var(--color-text-subtle);font-size:0.74rem;text-transform:uppercase;letter-spacing:0.03em;margin-bottom:0.2rem;';
        t.textContent = type;
        wrap.appendChild(t);
    }

    const desc = document.createElement('div');
    if (spec.Description) {
        desc.style.cssText = 'color:var(--color-text-muted);';
        desc.textContent = spec.Description;
    } else {
        desc.style.cssText = 'color:var(--color-text-faint);font-style:italic;';
        desc.textContent = 'No description.';
    }
    wrap.appendChild(desc);

    return wrap;
});

// QuestTokenPicker
//
// Usage:
//   QuestTokenPicker.pick((token, label) => {
//     // token: e.g. "1001-start"
//     // label: e.g. "#1001 The Lost Artifact - start"
//   });
//
// Optional: pass excludeQuestId to filter out the quest currently being edited
// (prevents a quest from referencing its own tokens).
//   QuestTokenPicker.pick(callback, { excludeQuestId: currentQuestId });

const QuestTokenPicker = (() => {
    function pick(onPicked, opts) {
        const exclude = opts && opts.excludeQuestId;

        Picker.open({
            ...PickerConfigs.quests,
            title:  'Select Quest',
            filter: exclude != null ? (q => q.QuestId !== exclude) : undefined,
            onSelect: (quest) => {
                const steps = (quest.Steps || []);

                // Build step items - always include start/end even if Steps is sparse
                const stepItems = steps.map(s => ({
                    Id:          s.Id,
                    Description: s.Description || '',
                }));

                Picker.open({
                    title:   'Select Step - #' + quest.QuestId + ' ' + quest.Name,
                    source:  async () => stepItems,
                    idKey:   'Id',
                    columns: [
                        { key: 'Id',          label: 'Step',        width: '8rem', mono: true },
                        { key: 'Description', label: 'Description', flex: true },
                    ],
                    searchKeys: ['Id', 'Description'],
                    onSelect: (step) => {
                        const token = quest.QuestId + '-' + step.Id;
                        const label = '#' + quest.QuestId + ' ' + quest.Name + ' \u2014 ' + step.Id;
                        onPicked(token, label);
                    },
                });
            },
        });
    }

    return { pick };
})();

// UserPicker - search-driven user picker backed by /admin/api/v1/users/search.
//
// Unlike Picker, this modal does not pre-load all users. Instead it fires a
// debounced search request as the user types and renders live results.
//
// Usage:
//   UserPicker.open({
//     onSelect: (user) => {
//       // user: { user_id, username, role, email }
//     },
//   });

const UserPicker = (() => {
    'use strict';

    let overlay = null;
    let _triggerEl = null;

    function injectStyles() {
        if (document.getElementById('user-picker-styles')) return;
        const style = document.createElement('style');
        style.id = 'user-picker-styles';
        style.textContent = `
            .up-overlay {
                position: fixed; inset: 0; background: var(--color-overlay);
                display: flex; align-items: center; justify-content: center;
                z-index: 9999;
            }
            .up-modal {
                background: var(--color-surface); border-radius: 8px; box-shadow: 0 8px 32px var(--color-shadow);
                width: 520px; max-width: 96vw; max-height: 80vh;
                display: flex; flex-direction: column; overflow: hidden;
            }
            .up-header {
                padding: 0.85rem 1rem 0.7rem; border-bottom: 1px solid var(--color-border);
                display: flex; align-items: center; justify-content: space-between; flex-shrink: 0;
            }
            .up-title { font-size: 1rem; font-weight: 700; color: var(--color-text); }
            .up-close {
                background: none; border: none; font-size: 1.25rem; cursor: pointer;
                color: var(--color-text-faint); line-height: 1; padding: 0 0.2rem;
            }
            .up-close:hover { color: var(--color-text); }
            .up-search-wrap { padding: 0.6rem 1rem; border-bottom: 1px solid var(--color-border-light); flex-shrink: 0; }
            .up-search {
                width: 100%; padding: 0.4rem 0.65rem; border: 1px solid var(--color-border-medium);
                border-radius: 4px; font-size: 0.875rem;
                background: var(--color-surface-raised); color: var(--color-text);
            }
            .up-search:focus { outline: 2px solid var(--color-focus); outline-offset: 1px; border-color: transparent; }
            .up-hint { font-size: 0.75rem; color: var(--color-text-placeholder); margin-top: 0.3rem; }
            .up-table-wrap { overflow-y: auto; flex: 1; background: var(--color-surface); }
            .up-table { width: 100%; border-collapse: collapse; font-size: 0.875rem; }
            .up-table thead { position: sticky; top: 0; background: var(--color-page-bg); z-index: 1; }
            .up-table th {
                padding: 0.4rem 0.75rem; text-align: left; font-size: 0.72rem;
                font-weight: 700; text-transform: uppercase; letter-spacing: 0.04em;
                color: var(--color-text-subtle); border-bottom: 1px solid var(--color-border-strong); white-space: nowrap;
            }
            .up-table td { padding: 0.45rem 0.75rem; border-bottom: 1px solid var(--color-border-faint); vertical-align: middle; color: var(--color-text); }
            .up-table tr:last-child td { border-bottom: none; }
            .up-table tbody tr { cursor: pointer; }
            .up-table tbody tr:hover { background: var(--color-row-hover); }
            .up-table tbody tr.up-highlighted { background: var(--color-row-hover); }
            .up-role-badge {
                display: inline-block; font-size: 0.7rem; padding: 0.1rem 0.4rem;
                border-radius: 3px; background: var(--color-chip-bg); color: var(--color-chip-text);
                font-weight: 600; text-transform: uppercase;
            }
            .up-role-badge.up-role-admin { background: var(--color-badge-hostile-bg); color: var(--color-badge-hostile-text); }
            .up-empty { padding: 2rem 1rem; text-align: center; color: var(--color-text-placeholder); font-size: 0.875rem; }
            .up-loading { padding: 2rem 1rem; text-align: center; color: var(--color-text-faint); font-size: 0.875rem; }
            .up-footer {
                padding: 0.65rem 1rem; border-top: 1px solid var(--color-border-light);
                display: flex; justify-content: flex-end; flex-shrink: 0;
                background: var(--color-surface);
            }
            .up-btn-cancel {
                padding: 0.4rem 1rem; border-radius: 4px; font-size: 0.85rem;
                font-weight: 600; cursor: pointer; background: var(--color-btn-cancel-bg);
                color: var(--color-btn-cancel-text); border: 1px solid var(--color-border-medium);
            }
            .up-btn-cancel:hover { background: var(--color-btn-cancel-hover); }
        `;
        document.head.appendChild(style);
    }

    function escHtml(s) {
        return String(s).replace(/&/g,'&amp;').replace(/</g,'&lt;').replace(/>/g,'&gt;').replace(/"/g,'&quot;');
    }

    function close() {
        if (overlay) { overlay.remove(); overlay = null; }
        if (_triggerEl) { _triggerEl.focus(); _triggerEl = null; }
    }

    function open({ onSelect }) {
        injectStyles();
        close();

        _triggerEl = document.activeElement || null;

        overlay = document.createElement('div');
        overlay.className = 'up-overlay';
        overlay.setAttribute('role', 'dialog');
        overlay.setAttribute('aria-modal', 'true');
        overlay.setAttribute('aria-label', 'Select User');
        overlay.addEventListener('click', e => { if (e.target === overlay) close(); });

        const modal = document.createElement('div');
        modal.className = 'up-modal';

        const header = document.createElement('div');
        header.className = 'up-header';
        const titleEl = document.createElement('span');
        titleEl.className = 'up-title';
        titleEl.textContent = 'Select User';
        const closeBtn = document.createElement('button');
        closeBtn.className = 'up-close';
        closeBtn.textContent = '\u00d7';
        closeBtn.setAttribute('aria-label', 'Close');
        closeBtn.addEventListener('click', close);
        header.appendChild(titleEl);
        header.appendChild(closeBtn);

        const searchWrap = document.createElement('div');
        searchWrap.className = 'up-search-wrap';
        const searchInput = document.createElement('input');
        searchInput.type = 'search';
        searchInput.className = 'up-search';
        searchInput.placeholder = 'Type a username to search\u2026';
        searchInput.setAttribute('aria-label', 'Search users');

        // Role filter buttons
        const roleFilterWrap = document.createElement('div');
        roleFilterWrap.style.cssText = 'display:flex;gap:0.4rem;margin-top:0.45rem;flex-wrap:wrap;';
        const roleFilters = [
            { label: 'All Admins', role: 'admin' },
            { label: 'All Mods',   role: 'mod'   },
        ];
        let activeRole = null;
        roleFilters.forEach(({ label, role }) => {
            const btn = document.createElement('button');
            btn.type = 'button';
            btn.textContent = label;
            btn.dataset.role = role;
            btn.style.cssText = 'padding:0.2rem 0.65rem;border-radius:999px;font-size:0.78rem;font-weight:600;cursor:pointer;border:1px solid var(--color-border-medium);background:var(--color-surface-raised);color:var(--color-text-muted);transition:background 0.12s,color 0.12s,border-color 0.12s;';
            btn.addEventListener('click', () => {
                if (activeRole === role) {
                    // Deactivate — go back to search mode
                    activeRole = null;
                    btn.style.background = 'var(--color-surface-raised)';
                    btn.style.color = 'var(--color-text-muted)';
                    btn.style.borderColor = 'var(--color-border-medium)';
                    tableWrap.innerHTML = '<div class="up-empty">Start typing to search for a user.</div>';
                    visibleUsers = [];
                    searchInput.focus();
                } else {
                    // Activate this role filter, deactivate others
                    activeRole = role;
                    roleFilterWrap.querySelectorAll('button').forEach(b => {
                        b.style.background = 'var(--color-surface-raised)';
                        b.style.color = 'var(--color-text-muted)';
                        b.style.borderColor = 'var(--color-border-medium)';
                    });
                    btn.style.background = 'var(--color-primary)';
                    btn.style.color = 'var(--color-primary-on)';
                    btn.style.borderColor = 'var(--color-primary)';
                    searchInput.value = '';
                    clearTimeout(debounceTimer);
                    doRoleSearch(role);
                }
            });
            roleFilterWrap.appendChild(btn);
        });

        const hint = document.createElement('div');
        hint.className = 'up-hint';
        hint.textContent = 'Type a username (min 2 chars) or a numeric user ID to search.';
        searchWrap.appendChild(searchInput);
        searchWrap.appendChild(roleFilterWrap);
        searchWrap.appendChild(hint);

        const tableWrap = document.createElement('div');
        tableWrap.className = 'up-table-wrap';
        tableWrap.innerHTML = '<div class="up-empty">Start typing to search for a user.</div>';

        const footer = document.createElement('div');
        footer.className = 'up-footer';
        const cancelBtn = document.createElement('button');
        cancelBtn.className = 'up-btn-cancel';
        cancelBtn.textContent = 'Cancel';
        cancelBtn.addEventListener('click', close);
        footer.appendChild(cancelBtn);

        modal.append(header, searchWrap, tableWrap, footer);
        overlay.appendChild(modal);
        document.body.appendChild(overlay);
        searchInput.focus();

        let highlightIdx = -1;
        let visibleUsers = [];
        let debounceTimer = null;

        function renderResults(users) {
            visibleUsers = users;
            highlightIdx = -1;

            if (!users.length) {
                tableWrap.innerHTML = '<div class="up-empty">No users found.</div>';
                return;
            }

            const table = document.createElement('table');
            table.className = 'up-table';

            const thead = document.createElement('thead');
            thead.innerHTML = '<tr><th style="width:4rem">ID</th><th>Username</th><th style="width:6rem">Role</th></tr>';
            table.appendChild(thead);

            const tbody = document.createElement('tbody');
            users.forEach((u, i) => {
                const tr = document.createElement('tr');
                tr.dataset.idx = i;
                const roleClass = (u.role || '').toLowerCase() === 'admin' ? ' up-role-admin' : '';
                tr.innerHTML =
                    '<td style="font-family:monospace">' + escHtml(String(u.user_id)) + '</td>' +
                    '<td><strong>' + escHtml(u.username || '') + '</strong>' +
                    (u.email ? '<br><span style="font-size:0.78rem;color:var(--color-text-faint)">' + escHtml(u.email) + '</span>' : '') +
                    '</td>' +
                    '<td><span class="up-role-badge' + roleClass + '">' + escHtml(u.role || 'user') + '</span></td>';
                tr.addEventListener('click', () => { close(); onSelect(u); });
                tbody.appendChild(tr);
            });
            table.appendChild(tbody);

            tableWrap.innerHTML = '';
            tableWrap.appendChild(table);
        }

        function setHighlight(idx) {
            const rows = tableWrap.querySelectorAll('tbody tr');
            if (highlightIdx >= 0 && rows[highlightIdx]) rows[highlightIdx].classList.remove('up-highlighted');
            highlightIdx = Math.max(0, Math.min(idx, visibleUsers.length - 1));
            if (rows[highlightIdx]) {
                rows[highlightIdx].classList.add('up-highlighted');
                rows[highlightIdx].scrollIntoView({ block: 'nearest' });
            }
        }

        async function doSearch(q) {
            activeRole = null;
            roleFilterWrap.querySelectorAll('button').forEach(b => {
                b.style.background = 'var(--color-surface-raised)';
                b.style.color = 'var(--color-text-muted)';
                b.style.borderColor = 'var(--color-border-medium)';
            });
            const trimmed = q.trim();
            const isNumeric = /^\d+$/.test(trimmed);
            if (trimmed.length < 2 && !isNumeric) {
                tableWrap.innerHTML = '<div class="up-empty">Start typing to search for a user.</div>';
                visibleUsers = [];
                return;
            }
            tableWrap.innerHTML = '<div class="up-loading">Searching\u2026</div>';
            const res = await AdminAPI.get('/admin/api/v1/users/search?name=' + encodeURIComponent(q.trim()), true);
            if (!res.ok) {
                tableWrap.innerHTML = '<div class="up-empty">Search failed: ' + escHtml(res.error || 'unknown error') + '</div>';
                return;
            }
            renderResults((res.data && res.data.data) || []);
        }

        async function doRoleSearch(role) {
            tableWrap.innerHTML = '<div class="up-loading">Loading\u2026</div>';
            const res = await AdminAPI.get('/admin/api/v1/users/search?role=' + encodeURIComponent(role), true);
            if (!res.ok) {
                tableWrap.innerHTML = '<div class="up-empty">Search failed: ' + escHtml(res.error || 'unknown error') + '</div>';
                return;
            }
            renderResults((res.data && res.data.data) || []);
        }

        searchInput.addEventListener('input', () => {
            clearTimeout(debounceTimer);
            debounceTimer = setTimeout(() => doSearch(searchInput.value), 200);
        });

        overlay.addEventListener('keydown', e => {
            if (e.key === 'Escape') { e.preventDefault(); close(); return; }
            if (e.key === 'ArrowDown') { e.preventDefault(); setHighlight(highlightIdx < 0 ? 0 : highlightIdx + 1); return; }
            if (e.key === 'ArrowUp')   { e.preventDefault(); setHighlight(highlightIdx <= 0 ? 0 : highlightIdx - 1); return; }
            if (e.key === 'Enter') {
                e.preventDefault();
                if (highlightIdx >= 0 && visibleUsers[highlightIdx]) {
                    close();
                    onSelect(visibleUsers[highlightIdx]);
                }
            }
        });
    }

    return { open, close };
})();

// CharacterPicker - search-driven character picker backed by
// /admin/api/v1/characters/search.
//
// Unlike Picker, this modal does not pre-load all characters. It fires a
// debounced search request as the user types and renders live results.
//
// Usage:
//   CharacterPicker.open({
//     onSelect: (result) => {
//       // result: { user_id, username, character_name }
//     },
//   });

const CharacterPicker = (() => {
    'use strict';

    let overlay = null;
    let _triggerEl = null;

    function escHtml(s) {
        return String(s).replace(/&/g,'&amp;').replace(/</g,'&lt;').replace(/>/g,'&gt;').replace(/"/g,'&quot;');
    }

    function close() {
        if (overlay) { overlay.remove(); overlay = null; }
        if (_triggerEl) { _triggerEl.focus(); _triggerEl = null; }
    }

    function open({ onSelect }) {
        // Ensure the up-* styles from UserPicker are present.
        if (typeof UserPicker !== 'undefined') {
            UserPicker.open({ onSelect: () => {} });
            UserPicker.close();
        }
        close();

        _triggerEl = document.activeElement || null;

        overlay = document.createElement('div');
        overlay.className = 'up-overlay';
        overlay.setAttribute('role', 'dialog');
        overlay.setAttribute('aria-modal', 'true');
        overlay.setAttribute('aria-label', 'Select Character');
        overlay.addEventListener('click', e => { if (e.target === overlay) close(); });

        const modal = document.createElement('div');
        modal.className = 'up-modal';

        const header = document.createElement('div');
        header.className = 'up-header';
        const titleEl = document.createElement('span');
        titleEl.className = 'up-title';
        titleEl.textContent = 'Select Character';
        const closeBtn = document.createElement('button');
        closeBtn.className = 'up-close';
        closeBtn.textContent = '\u00d7';
        closeBtn.setAttribute('aria-label', 'Close');
        closeBtn.addEventListener('click', close);
        header.appendChild(titleEl);
        header.appendChild(closeBtn);

        const searchWrap = document.createElement('div');
        searchWrap.className = 'up-search-wrap';
        const searchInput = document.createElement('input');
        searchInput.type = 'search';
        searchInput.className = 'up-search';
        searchInput.placeholder = 'Type a character name to search\u2026';
        searchInput.setAttribute('aria-label', 'Search characters');
        const hint = document.createElement('div');
        hint.className = 'up-hint';
        hint.textContent = 'Type a character name (min 2 chars) to search.';
        searchWrap.appendChild(searchInput);
        searchWrap.appendChild(hint);

        const tableWrap = document.createElement('div');
        tableWrap.className = 'up-table-wrap';
        tableWrap.innerHTML = '<div class="up-empty">Start typing to search for a character.</div>';

        const footer = document.createElement('div');
        footer.className = 'up-footer';
        const cancelBtn = document.createElement('button');
        cancelBtn.className = 'up-btn-cancel';
        cancelBtn.textContent = 'Cancel';
        cancelBtn.addEventListener('click', close);
        footer.appendChild(cancelBtn);

        modal.append(header, searchWrap, tableWrap, footer);
        overlay.appendChild(modal);
        document.body.appendChild(overlay);
        searchInput.focus();

        let highlightIdx = -1;
        let visibleResults = [];
        let debounceTimer = null;

        function renderResults(results) {
            visibleResults = results;
            highlightIdx = -1;

            if (!results.length) {
                tableWrap.innerHTML = '<div class="up-empty">No characters found.</div>';
                return;
            }

            const table = document.createElement('table');
            table.className = 'up-table';
            const thead = document.createElement('thead');
            thead.innerHTML = '<tr><th style="width:4rem">User ID</th><th>Character</th><th>Username</th></tr>';
            table.appendChild(thead);

            const tbody = document.createElement('tbody');
            results.forEach((r, i) => {
                const tr = document.createElement('tr');
                tr.dataset.idx = i;
                tr.innerHTML =
                    '<td style="font-family:monospace">' + escHtml(String(r.user_id)) + '</td>' +
                    '<td><strong>' + escHtml(r.character_name || '') + '</strong></td>' +
                    '<td style="color:var(--color-text-faint)">' + escHtml(r.username || '') + '</td>';
                tr.addEventListener('click', () => { close(); onSelect(r); });
                tbody.appendChild(tr);
            });
            table.appendChild(tbody);

            tableWrap.innerHTML = '';
            tableWrap.appendChild(table);
        }

        function setHighlight(idx) {
            const rows = tableWrap.querySelectorAll('tbody tr');
            if (highlightIdx >= 0 && rows[highlightIdx]) rows[highlightIdx].classList.remove('up-highlighted');
            highlightIdx = Math.max(0, Math.min(idx, visibleResults.length - 1));
            if (rows[highlightIdx]) {
                rows[highlightIdx].classList.add('up-highlighted');
                rows[highlightIdx].scrollIntoView({ block: 'nearest' });
            }
        }

        async function doSearch(q) {
            const trimmed = q.trim();
            if (trimmed.length < 2) {
                tableWrap.innerHTML = '<div class="up-empty">Start typing to search for a character.</div>';
                visibleResults = [];
                return;
            }
            tableWrap.innerHTML = '<div class="up-loading">Searching\u2026</div>';
            const res = await AdminAPI.get('/admin/api/v1/characters/search?name=' + encodeURIComponent(trimmed), true);
            if (!res.ok) {
                tableWrap.innerHTML = '<div class="up-empty">Search failed: ' + escHtml(res.error || 'unknown error') + '</div>';
                return;
            }
            renderResults((res.data && res.data.data) || []);
        }

        searchInput.addEventListener('input', () => {
            clearTimeout(debounceTimer);
            debounceTimer = setTimeout(() => doSearch(searchInput.value), 200);
        });

        overlay.addEventListener('keydown', e => {
            if (e.key === 'Escape') { e.preventDefault(); close(); return; }
            if (e.key === 'ArrowDown') { e.preventDefault(); setHighlight(highlightIdx < 0 ? 0 : highlightIdx + 1); return; }
            if (e.key === 'ArrowUp')   { e.preventDefault(); setHighlight(highlightIdx <= 0 ? 0 : highlightIdx - 1); return; }
            if (e.key === 'Enter') {
                e.preventDefault();
                if (highlightIdx >= 0 && visibleResults[highlightIdx]) {
                    close();
                    onSelect(visibleResults[highlightIdx]);
                }
            }
        });
    }

    return { open, close };
})();
