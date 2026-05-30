/**
 * window-killstats.js
 *
 * Virtual window: Kill Stats - right dock, tabbed, off by default.
 *
 * Tabs:
 *   Mobs   - kill counts by mob name with totals and K/D ratio
 *   Races  - kill counts grouped by race
 *   Areas  - kill counts grouped by zone/area
 *   PvP    - player kill counts with K/D ratio
 *
 * Responds to GMCP namespace:
 *   Char.Kills  - kill/death stats
 *   Char        - full character update
 *
 * Reads: Client.GMCPStructs.Char.Kills
 *
 * Disabled by default (offOnLoad: true).
 */

'use strict';

(function() {

    injectStyles(`
        /* ---- shell ---- */
        #ks-window {
            height: 100%;
            display: flex;
            flex-direction: column;
            background: var(--t-bg);
        }

        /* ---- tab bar ---- */
        #ks-window .ks-tab-bar {
            display: flex;
            flex-shrink: 0;
            border-bottom: 1px solid var(--t-border);
        }

        #ks-window .ks-tab-btn {
            flex: 1;
            padding: 5px 4px;
            background: var(--t-bg-surface);
            border: none;
            cursor: pointer;
            font: inherit;
            font-size: 0.7em;
            color: var(--t-text-secondary);
            text-transform: uppercase;
            letter-spacing: 0.04em;
            transition: background 0.15s, color 0.15s;
            border-right: 1px solid var(--t-border);
        }

        #ks-window .ks-tab-btn:last-child { border-right: none; }

        #ks-window .ks-tab-btn:hover {
            background: var(--t-border);
            color: var(--t-text);
        }

        #ks-window .ks-tab-btn.active {
            background: var(--t-bg);
            color: var(--t-text);
            border-bottom: 2px solid var(--t-accent);
        }

        /* ---- tab panels ---- */
        #ks-window .ks-tab-panel {
            display: none;
            flex: 1;
            flex-direction: column;
            overflow: hidden;
        }

        #ks-window .ks-tab-panel.active {
            display: flex;
        }

        /* ---- summary bar ---- */
        .ks-summary {
            display: flex;
            gap: 0;
            flex-shrink: 0;
            border-bottom: 1px solid var(--t-border);
        }

        .ks-summary-cell {
            flex: 1;
            display: flex;
            flex-direction: column;
            align-items: center;
            padding: 5px 4px;
            border-right: 1px solid var(--t-border);
        }

        .ks-summary-cell:last-child { border-right: none; }

        .ks-summary-label {
            font-size: 0.6em;
            color: var(--t-text-secondary);
            text-transform: uppercase;
            letter-spacing: 0.04em;
        }

        .ks-summary-value {
            font-size: 0.9em;
            color: var(--t-text);
            font-weight: bold;
        }

        .ks-summary-value.kd-good  { color: var(--t-kd-good); }
        .ks-summary-value.kd-bad   { color: var(--t-kd-bad); }
        .ks-summary-value.kd-even  { color: var(--t-kd-even); }

        /* ---- list area ---- */
        .ks-list {
            flex: 1;
            overflow-y: auto;
        }

        .ks-list::-webkit-scrollbar       { width: 4px; }
        .ks-list::-webkit-scrollbar-track  { background: var(--t-scrollbar-track); }
        .ks-list::-webkit-scrollbar-thumb  { background: var(--t-accent-dim); border-radius: 2px; }

        /* ---- list rows ---- */
        .ks-row {
            display: flex;
            align-items: center;
            padding: 4px 8px;
            border-bottom: 1px solid var(--t-border-faint);
            gap: 6px;
        }

        .ks-row:last-child { border-bottom: none; }

        .ks-row:hover { background: var(--t-bg-surface); }

        .ks-row-name {
            flex: 1;
            font-size: 0.78em;
            color: var(--t-text);
            overflow: hidden;
            text-overflow: ellipsis;
            white-space: nowrap;
        }

        .ks-row-count {
            font-size: 0.78em;
            color: var(--t-accent);
            font-weight: bold;
            width: 36px;
            text-align: right;
            flex-shrink: 0;
        }

        .ks-row-pct {
            font-size: 0.72em;
            color: var(--t-text-secondary);
            width: 38px;
            text-align: right;
            flex-shrink: 0;
        }

        /* ---- bar fill ---- */
        .ks-bar-track {
            width: 52px;
            height: 5px;
            background: var(--t-bg-row);
            border-radius: 3px;
            overflow: hidden;
            flex-shrink: 0;
        }

        .ks-bar-fill {
            height: 100%;
            border-radius: 3px;
            background: linear-gradient(to right, var(--t-progress-from), var(--t-progress-to));
        }

        /* ---- empty state ---- */
        .ks-empty {
            padding: 16px 10px;
            color: var(--t-text-dim);
            font-size: 0.76em;
            font-style: italic;
            text-align: center;
        }

        /* ---- col headers ---- */
        .ks-col-header {
            display: flex;
            align-items: center;
            padding: 3px 8px;
            background: var(--t-bg-col-header);
            border-bottom: 1px solid var(--t-border);
            flex-shrink: 0;
            gap: 6px;
        }

        .ks-col-header-name {
            flex: 1;
            font-size: 0.62em;
            color: var(--t-text-heading);
            text-transform: uppercase;
            letter-spacing: 0.07em;
        }

        .ks-col-header-count {
            font-size: 0.62em;
            color: var(--t-text-heading);
            text-transform: uppercase;
            letter-spacing: 0.07em;
            width: 36px;
            text-align: right;
            flex-shrink: 0;
        }

        .ks-col-header-pct {
            font-size: 0.62em;
            color: var(--t-text-heading);
            text-transform: uppercase;
            letter-spacing: 0.07em;
            width: 38px;
            text-align: right;
            flex-shrink: 0;
        }

        .ks-col-header-bar {
            width: 52px;
            flex-shrink: 0;
        }

        .ks-row-elite {
            font-size: 0.78em;
            color: var(--t-text-secondary);
            width: 42px;
            text-align: right;
            flex-shrink: 0;
        }

        .ks-row-elite.has-elites {
            color: #dca000;
            font-weight: bold;
        }

        .ks-col-header-elite {
            font-size: 0.62em;
            color: var(--t-text-heading);
            text-transform: uppercase;
            letter-spacing: 0.07em;
            width: 42px;
            text-align: right;
            flex-shrink: 0;
        }
    `);

    // -----------------------------------------------------------------------
    // Tab switching
    // -----------------------------------------------------------------------
    function makeTabSwitcher(root) {
        const btns   = root.querySelectorAll('.ks-tab-btn');
        const panels = root.querySelectorAll('.ks-tab-panel');
        btns.forEach(function(btn) {
            btn.addEventListener('click', function() {
                btns.forEach(function(b)   { b.classList.remove('active'); });
                panels.forEach(function(p) { p.classList.remove('active'); });
                btn.classList.add('active');
                root.querySelector('#' + btn.dataset.panel).classList.add('active');
            });
        });
    }

    // -----------------------------------------------------------------------
    // DOM factory
    // -----------------------------------------------------------------------
    function createDOM() {
        const el = document.createElement('div');
        el.id = 'ks-window';

        el.innerHTML =
            '<div class="ks-tab-bar">' +
                '<button class="ks-tab-btn active" data-panel="ks-mobs">Mobs</button>' +
                '<button class="ks-tab-btn"        data-panel="ks-races">Races</button>' +
                '<button class="ks-tab-btn"        data-panel="ks-areas">Areas</button>' +
                '<button class="ks-tab-btn"        data-panel="ks-pvp">PvP</button>' +
            '</div>' +

            /* Mobs tab */
            '<div class="ks-tab-panel active" id="ks-mobs">' +
                '<div class="ks-summary" id="ks-mob-summary">' +
                    '<div class="ks-summary-cell"><span class="ks-summary-label">Kills</span><span class="ks-summary-value" id="ks-mob-total">-</span></div>' +
                    '<div class="ks-summary-cell"><span class="ks-summary-label">Elites</span><span class="ks-summary-value" id="ks-mob-elites">-</span></div>' +
                    '<div class="ks-summary-cell"><span class="ks-summary-label">Deaths</span><span class="ks-summary-value" id="ks-mob-deaths">-</span></div>' +
                    '<div class="ks-summary-cell"><span class="ks-summary-label">K/D</span><span class="ks-summary-value" id="ks-mob-kd">-</span></div>' +
                '</div>' +
                '<div class="ks-col-header">' +
                    '<span class="ks-col-header-name">Mob</span>' +
                    '<span class="ks-col-header-count">Kills</span>' +
                    '<span class="ks-col-header-elite">Elites</span>' +
                    '<span class="ks-col-header-pct">%</span>' +
                    '<span class="ks-col-header-bar"></span>' +
                '</div>' +
                '<div class="ks-list" id="ks-mob-list"></div>' +
            '</div>' +

            /* Races tab */
            '<div class="ks-tab-panel" id="ks-races">' +
                '<div class="ks-col-header">' +
                    '<span class="ks-col-header-name">Race</span>' +
                    '<span class="ks-col-header-count">Kills</span>' +
                    '<span class="ks-col-header-pct">%</span>' +
                    '<span class="ks-col-header-bar"></span>' +
                '</div>' +
                '<div class="ks-list" id="ks-race-list"></div>' +
            '</div>' +

            /* Areas tab */
            '<div class="ks-tab-panel" id="ks-areas">' +
                '<div class="ks-col-header">' +
                    '<span class="ks-col-header-name">Area</span>' +
                    '<span class="ks-col-header-count">Kills</span>' +
                    '<span class="ks-col-header-pct">%</span>' +
                    '<span class="ks-col-header-bar"></span>' +
                '</div>' +
                '<div class="ks-list" id="ks-area-list"></div>' +
            '</div>' +

            /* PvP tab */
            '<div class="ks-tab-panel" id="ks-pvp">' +
                '<div class="ks-summary">' +
                    '<div class="ks-summary-cell"><span class="ks-summary-label">Kills</span><span class="ks-summary-value" id="ks-pvp-total">-</span></div>' +
                    '<div class="ks-summary-cell"><span class="ks-summary-label">Deaths</span><span class="ks-summary-value" id="ks-pvp-deaths">-</span></div>' +
                    '<div class="ks-summary-cell"><span class="ks-summary-label">K/D</span><span class="ks-summary-value" id="ks-pvp-kd">-</span></div>' +
                '</div>' +
                '<div class="ks-col-header">' +
                    '<span class="ks-col-header-name">Player</span>' +
                    '<span class="ks-col-header-count">Kills</span>' +
                    '<span class="ks-col-header-pct">%</span>' +
                    '<span class="ks-col-header-bar"></span>' +
                '</div>' +
                '<div class="ks-list" id="ks-pvp-list"></div>' +
            '</div>';

        document.body.appendChild(el);
        makeTabSwitcher(el);
        return el;
    }

    // -----------------------------------------------------------------------
    // VirtualWindow
    // -----------------------------------------------------------------------
    const win = new VirtualWindow('KillStats', {
        dock:          'right',
        defaultDocked: true,
        dockedHeight:  260,
        offOnLoad:     true,
        factory() {
            const el = createDOM();
            Client.GMCPRequest('Char.Kills');
            requestAnimationFrame(function() { update(); });
            return {
                title:      'Kill Stats',
                mount:      el,
                background: 'var(--t-bg)',
                border:     1,
                x:          'right',
                y:          0,
                width:      363,
                height:     300,
                header:     20,
                bottom:     60,
            };
        },
    });

    // -----------------------------------------------------------------------
    // Helpers
    // -----------------------------------------------------------------------
    function kdClass(ratio) {
        if (ratio > 1)  { return 'kd-good'; }
        if (ratio < 1)  { return 'kd-bad'; }
        return 'kd-even';
    }

    function fmtKD(ratio) {
        if (ratio === undefined || ratio === null) { return '-'; }
        return ratio.toFixed(2) + ':1';
    }

    function renderList(listEl, mapObj, total) {
        listEl.innerHTML = '';

        if (!mapObj || Object.keys(mapObj).length === 0) {
            listEl.innerHTML = '<div class="ks-empty">No data yet.</div>';
            return;
        }

        const entries = Object.entries(mapObj);
        entries.sort(function(a, b) { return b[1] - a[1]; });

        const maxVal = entries[0][1];

        entries.forEach(function(entry) {
            const name  = entry[0];
            const count = entry[1];
            const pct   = total > 0 ? (count / total * 100) : 0;
            const barW  = maxVal > 0 ? Math.round(count / maxVal * 100) : 0;

            const row = document.createElement('div');
            row.className = 'ks-row';
            row.innerHTML =
                '<span class="ks-row-name">' + name + '</span>' +
                '<span class="ks-row-count">' + count + '</span>' +
                '<span class="ks-row-pct">' + pct.toFixed(1) + '%</span>' +
                '<div class="ks-bar-track"><div class="ks-bar-fill" style="width:' + barW + '%"></div></div>';
            listEl.appendChild(row);
        });
    }

    function renderMobList(listEl, mapObj, eliteMap, total) {
        listEl.innerHTML = '';

        if (!mapObj || Object.keys(mapObj).length === 0) {
            listEl.innerHTML = '<div class="ks-empty">No data yet.</div>';
            return;
        }

        const entries = Object.entries(mapObj);
        entries.sort(function(a, b) { return b[1] - a[1]; });

        const maxVal = entries[0][1];
        const elite  = eliteMap || {};

        entries.forEach(function(entry) {
            const name      = entry[0];
            const count     = entry[1];
            const eliteCt   = elite[name] || 0;
            const pct       = total > 0 ? (count / total * 100) : 0;
            const barW      = maxVal > 0 ? Math.round(count / maxVal * 100) : 0;
            const eliteCls  = eliteCt > 0 ? ' has-elites' : '';
            const eliteTxt  = eliteCt > 0 ? String(eliteCt) : '-';

            const row = document.createElement('div');
            row.className = 'ks-row';
            row.innerHTML =
                '<span class="ks-row-name">' + name + '</span>' +
                '<span class="ks-row-count">' + count + '</span>' +
                '<span class="ks-row-elite' + eliteCls + '">' + eliteTxt + '</span>' +
                '<span class="ks-row-pct">' + pct.toFixed(1) + '%</span>' +
                '<div class="ks-bar-track"><div class="ks-bar-fill" style="width:' + barW + '%"></div></div>';
            listEl.appendChild(row);
        });
    }

    function renderPvpList(listEl, playersObj, total) {
        listEl.innerHTML = '';

        if (!playersObj || Object.keys(playersObj).length === 0) {
            listEl.innerHTML = '<div class="ks-empty">No PvP kills recorded.</div>';
            return;
        }

        const entries = Object.entries(playersObj).map(function(e) {
            return [e[0], e[1].count || 0];
        });
        entries.sort(function(a, b) { return b[1] - a[1]; });

        const maxVal = entries[0][1];

        entries.forEach(function(entry) {
            const name  = entry[0];
            const count = entry[1];
            const pct   = total > 0 ? (count / total * 100) : 0;
            const barW  = maxVal > 0 ? Math.round(count / maxVal * 100) : 0;

            const row = document.createElement('div');
            row.className = 'ks-row';
            row.innerHTML =
                '<span class="ks-row-name">' + name + '</span>' +
                '<span class="ks-row-count">' + count + '</span>' +
                '<span class="ks-row-pct">' + pct.toFixed(1) + '%</span>' +
                '<div class="ks-bar-track"><div class="ks-bar-fill" style="width:' + barW + '%"></div></div>';
            listEl.appendChild(row);
        });
    }

    // -----------------------------------------------------------------------
    // Update
    // -----------------------------------------------------------------------
    function update() {
        const kills = Client.GMCPStructs.Char && Client.GMCPStructs.Char.Kills;
        if (!kills) { return; }

        win.open();
        if (!win.isOpen()) { return; }

        const mob = kills.mob || {};
        const pvp = kills.pvp || {};

        /* Mob summary */
        const mobTotal  = document.getElementById('ks-mob-total');
        const mobElites = document.getElementById('ks-mob-elites');
        const mobDeaths = document.getElementById('ks-mob-deaths');
        const mobKD     = document.getElementById('ks-mob-kd');
        if (mobTotal)  { mobTotal.textContent  = mob.total  !== undefined ? mob.total  : '0'; }
        if (mobElites) { mobElites.textContent = mob.elite_kills !== undefined ? mob.elite_kills : '0'; }
        if (mobDeaths) { mobDeaths.textContent = mob.deaths !== undefined ? mob.deaths : '0'; }
        if (mobKD) {
            mobKD.textContent = fmtKD(mob.kd_ratio);
            mobKD.className   = 'ks-summary-value ' + kdClass(mob.kd_ratio);
        }

        /* Mob lists */
        const mobList  = document.getElementById('ks-mob-list');
        const raceList = document.getElementById('ks-race-list');
        const areaList = document.getElementById('ks-area-list');
        if (mobList)  { renderMobList(mobList, mob.by_name, mob.by_elite, mob.total); }
        if (raceList) { renderList(raceList, mob.by_race, mob.total); }
        if (areaList) { renderList(areaList, mob.by_area, mob.total); }

        /* PvP summary */
        const pvpTotal  = document.getElementById('ks-pvp-total');
        const pvpDeaths = document.getElementById('ks-pvp-deaths');
        const pvpKD     = document.getElementById('ks-pvp-kd');
        if (pvpTotal)  { pvpTotal.textContent  = pvp.total  !== undefined ? pvp.total  : '0'; }
        if (pvpDeaths) { pvpDeaths.textContent = pvp.deaths !== undefined ? pvp.deaths : '0'; }
        if (pvpKD) {
            pvpKD.textContent = fmtKD(pvp.kd_ratio);
            pvpKD.className   = 'ks-summary-value ' + kdClass(pvp.kd_ratio);
        }

        /* PvP list */
        const pvpList = document.getElementById('ks-pvp-list');
        if (pvpList) { renderPvpList(pvpList, pvp.players, pvp.total); }
    }

    // -----------------------------------------------------------------------
    // Registration
    // -----------------------------------------------------------------------
    VirtualWindows.register({
        window:       win,
        gmcpHandlers: ['Char.Kills', 'Char'],
        onGMCP() { update(); },
    });

})();
