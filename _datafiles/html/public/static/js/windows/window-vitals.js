/**
 * window-vitals.js
 *
 * Virtual window: Vitals (HP / MP bars).
 *
 * Responds to GMCP namespaces:
 *   Char.Vitals  - direct vitals update
 *   Char         - top-level Char update; delegates to Char.Vitals if present
 *
 * Reads: Client.GMCPStructs.Char.Vitals
 */

'use strict';

(function() {

    const SEGMENT_COUNT = 20;

    injectStyles(`
        #vitals-bars {
            height: 100%;
            display: flex;
            flex-direction: column;
            justify-content: center;
            gap: 6px;
            padding: 4px 6px;
            box-sizing: border-box;
        }

        .vitals-row {
            display: flex;
            flex-direction: column;
            gap: 2px;
        }

        .vitals-label-row {
            display: flex;
            align-items: baseline;
            justify-content: space-between;
            padding: 0 1px;
        }

        .vitals-label {
            font-family: monospace;
            font-size: 0.7em;
            font-weight: bold;
            letter-spacing: 0.08em;
            text-transform: uppercase;
            color: var(--t-text-secondary);
        }

        .vitals-value {
            font-family: monospace;
            font-size: 0.72em;
            color: var(--t-text-muted);
            letter-spacing: 0.04em;
        }

        .vitals-track {
            display: flex;
            gap: 2px;
            height: 14px;
            align-items: stretch;
        }

        .vitals-segment {
            flex: 1;
            border-radius: 2px;
            transition: background 0.25s ease, box-shadow 0.25s ease, opacity 0.25s ease;
            position: relative;
        }

        /* HP segments - filled color driven by fill level */
        .vitals-segment.hp-filled-high {
            background: var(--t-hp-high);
            box-shadow: 0 0 4px color-mix(in srgb, var(--t-hp-high) 50%, transparent);
        }

        .vitals-segment.hp-filled-mid {
            background: var(--t-hp-mid);
            box-shadow: 0 0 4px color-mix(in srgb, var(--t-hp-mid) 40%, transparent);
        }

        .vitals-segment.hp-filled-low {
            background: var(--t-hp-low);
            box-shadow: 0 0 5px color-mix(in srgb, var(--t-hp-low) 55%, transparent);
        }

        /* Mana segments */
        .vitals-segment.mp-filled {
            background: linear-gradient(to bottom, var(--t-mana-to), var(--t-mana-from));
            box-shadow: 0 0 4px color-mix(in srgb, var(--t-mana-to) 45%, transparent);
        }

        /* Empty segments */
        .vitals-segment.seg-empty {
            background: var(--t-bar-empty);
            box-shadow: none;
            opacity: 0.55;
        }

        /* Inset groove effect on empty */
        .vitals-segment.seg-empty::after {
            content: '';
            display: block;
            height: 100%;
            border-radius: 2px;
            background: linear-gradient(to bottom, rgba(0,0,0,0.3) 0%, transparent 60%);
        }

        /* Pulse animation on low HP */
        @keyframes vitals-pulse-low {
            0%   { opacity: 1; }
            50%  { opacity: 0.6; }
            100% { opacity: 1; }
        }

        .vitals-track.hp-critical .vitals-segment.hp-filled-low {
            animation: vitals-pulse-low 1.1s ease-in-out infinite;
        }
    `);

    // -----------------------------------------------------------------------
    // DOM factory
    // -----------------------------------------------------------------------
    function makeSegments(count, trackClass) {
        const track = document.createElement('div');
        track.className = 'vitals-track ' + trackClass;
        for (let i = 0; i < count; i++) {
            const seg = document.createElement('div');
            seg.className = 'vitals-segment seg-empty';
            track.appendChild(seg);
        }
        return track;
    }

    function createDOM() {
        const container = document.createElement('div');
        container.id = 'vitals-bars';

        // HP row
        const hpRow = document.createElement('div');
        hpRow.className = 'vitals-row';

        const hpLabelRow = document.createElement('div');
        hpLabelRow.className = 'vitals-label-row';
        hpLabelRow.innerHTML =
            '<span class="vitals-label">HP</span>' +
            '<span class="vitals-value" id="vitals-hp-value">-- / --</span>';

        const hpTrack = makeSegments(SEGMENT_COUNT, 'hp-track');
        hpTrack.id = 'vitals-hp-track';

        hpRow.appendChild(hpLabelRow);
        hpRow.appendChild(hpTrack);

        // MP row
        const mpRow = document.createElement('div');
        mpRow.className = 'vitals-row';

        const mpLabelRow = document.createElement('div');
        mpLabelRow.className = 'vitals-label-row';
        mpLabelRow.innerHTML =
            '<span class="vitals-label">MP</span>' +
            '<span class="vitals-value" id="vitals-mp-value">-- / --</span>';

        const mpTrack = makeSegments(SEGMENT_COUNT, 'mp-track');
        mpTrack.id = 'vitals-mp-track';

        mpRow.appendChild(mpLabelRow);
        mpRow.appendChild(mpTrack);

        container.appendChild(hpRow);
        container.appendChild(mpRow);

        document.body.appendChild(container);
        return container;
    }

    // -----------------------------------------------------------------------
    // VirtualWindow instance
    // -----------------------------------------------------------------------
    const win = new VirtualWindow('Vitals', {
        dock:          'left',
        defaultDocked: true,
        dockedHeight:  100,
        factory() {
            const el = createDOM();
            return {
                title:      'Vitals',
                mount:      el,
                background: 'var(--t-bg)',
                border:     1,
                x:          0,
                y:          0,
                width:      300,
                height:     20 + 100,
                header:     20,
                bottom:     72,
            };
        },
    });

    // -----------------------------------------------------------------------
    // Segment update helpers
    // -----------------------------------------------------------------------
    function hpClass(pct) {
        if (pct > 60) return 'hp-filled-high';
        if (pct > 25) return 'hp-filled-mid';
        return 'hp-filled-low';
    }

    function updateTrack(trackEl, filledCount, filledClass, isCritical) {
        const segs = trackEl.children;
        for (let i = 0; i < segs.length; i++) {
            const seg = segs[i];
            if (i < filledCount) {
                seg.className = 'vitals-segment ' + filledClass;
            } else {
                seg.className = 'vitals-segment seg-empty';
            }
        }
        if (isCritical !== undefined) {
            trackEl.classList.toggle('hp-critical', isCritical);
        }
    }

    // -----------------------------------------------------------------------
    // Update logic
    // -----------------------------------------------------------------------
    function updateBars() {
        const vitals = Client.GMCPStructs.Char && Client.GMCPStructs.Char.Vitals;
        if (!vitals) return;

        win.open();
        if (!win.isOpen()) return;

        const hpPct = Math.max(0, Math.min(100, vitals.hp_max > 0 ? Math.floor(vitals.hp / vitals.hp_max * 100) : 0));
        const mpPct = Math.max(0, Math.min(100, vitals.sp_max > 0 ? Math.floor(vitals.sp / vitals.sp_max * 100) : 0));

        const hpFilled = Math.round(hpPct / 100 * SEGMENT_COUNT);
        const mpFilled = Math.round(mpPct / 100 * SEGMENT_COUNT);

        const hpTrack = document.getElementById('vitals-hp-track');
        const mpTrack = document.getElementById('vitals-mp-track');
        const hpVal   = document.getElementById('vitals-hp-value');
        const mpVal   = document.getElementById('vitals-mp-value');

        if (hpTrack) updateTrack(hpTrack, hpFilled, hpClass(hpPct), hpPct <= 25);
        if (mpTrack) updateTrack(mpTrack, mpFilled, 'mp-filled');
        if (hpVal)  hpVal.textContent  = vitals.hp + ' / ' + vitals.hp_max;
        if (mpVal)  mpVal.textContent  = vitals.sp + ' / ' + vitals.sp_max;
    }

    // -----------------------------------------------------------------------
    // Registration
    // -----------------------------------------------------------------------
    VirtualWindows.register({
        window:       win,
        gmcpHandlers: ['Char.Vitals', 'Char'],
        onGMCP(namespace) {
            updateBars();
        },
    });

})();
