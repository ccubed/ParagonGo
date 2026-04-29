// DiceRollHelper - inline form builder for dice roll shorthand strings.
//
// Usage:
//   DiceRollHelper.attach(textInputElement)
//
// Adds a toggle button next to the input. When toggled, an inline form appears
// with controls for attacks, dice count, dice sides, bonus, and crit buff IDs.
// The form syncs bidirectionally with the text input.
//
// Full format: [attacks@]NdS[+bonus][#critBuffId,...]
//   e.g. 1d6, 2d4+1, 2@1d3+2, 1d6#4, 2@1d3+2#1,2,3

var DiceRollHelper = (function() {

    var CSS_ID = 'dice-roll-helper-styles';

    function injectStyles() {
        if (document.getElementById(CSS_ID)) return;
        var style = document.createElement('style');
        style.id = CSS_ID;
        style.textContent = [
            '.drh-toggle {',
            '  padding:0.35rem 0.55rem; border:1px solid var(--color-border-medium); border-radius:4px;',
            '  background:var(--color-surface-raised); cursor:pointer; font-size:0.82rem; line-height:1;',
            '  flex-shrink:0; font-family:monospace; font-weight:600; color:var(--color-text-muted);',
            '}',
            '.drh-toggle:hover { background:var(--color-row-hover); border-color:var(--color-accent-link); }',
            '.drh-toggle.active { background:var(--color-btn-primary-bg); color:var(--color-btn-primary-text); border-color:var(--color-btn-primary-bg); }',
            '.drh-panel {',
            '  display:none; margin-top:0.5rem; padding:0.7rem 0.8rem;',
            '  background:var(--color-surface-alt); border:1px solid var(--color-border); border-radius:6px;',
            '}',
            '.drh-panel.open { display:block; }',
            '.drh-row { display:flex; gap:0.35rem; align-items:center; }',
            '.drh-group { display:flex; flex-direction:column; gap:0.15rem; flex-shrink:0; }',
            '.drh-group label {',
            '  font-size:0.68rem !important; font-weight:600; color:var(--color-text-faint);',
            '  text-transform:uppercase; letter-spacing:0.04em; white-space:nowrap;',
            '}',
            '.drh-group input[type="number"] {',
            '  width:52px; padding:0.3rem 0.3rem; border:1px solid var(--color-border-medium);',
            '  border-radius:4px; font-size:0.85rem; font-family:inherit; text-align:center;',
            '  background:var(--color-surface); color:var(--color-text);',
            '}',
            '.drh-sep {',
            '  font-size:1rem; font-weight:700; color:var(--color-text-secondary); padding-top:1rem;',
            '  user-select:none; flex-shrink:0;',
            '}',
            '.drh-preview {',
            '  margin-top:0.4rem; font-size:0.82rem; color:var(--color-text-muted); font-family:monospace;',
            '}',
            '.drh-preview code {',
            '  background:var(--color-chip-bg); padding:0.15rem 0.4rem; border-radius:3px;',
            '  font-weight:600; color:var(--color-chip-text);',
            '}',
            '.drh-crit-row {',
            '  display:flex; align-items:center; gap:0.4rem; margin-top:0.45rem; flex-wrap:wrap;',
            '}',
            '.drh-crit-row .drh-label {',
            '  font-size:0.68rem; font-weight:600; color:var(--color-text-faint);',
            '  text-transform:uppercase; letter-spacing:0.04em; flex-shrink:0;',
            '}',
            '.drh-crit-chips { display:flex; gap:0.3rem; flex-wrap:wrap; align-items:center; }',
            '.drh-chip {',
            '  display:inline-flex; align-items:center; gap:0.25rem;',
            '  background:var(--color-chip-bg); color:var(--color-chip-text); border-radius:3px;',
            '  padding:0.15rem 0.45rem; font-size:0.78rem; font-family:monospace;',
            '}',
            '.drh-chip button {',
            '  background:none; border:none; cursor:pointer; color:var(--color-text-faint);',
            '  font-size:0.9rem; line-height:1; padding:0;',
            '}',
            '.drh-chip button:hover { color:var(--color-danger); }',
            '.drh-add-buff {',
            '  font-size:0.75rem; padding:0.2rem 0.5rem; border:1px dashed var(--color-text-placeholder);',
            '  border-radius:4px; background:var(--color-surface); cursor:pointer; color:var(--color-text-muted);',
            '}',
            '.drh-add-buff:hover { border-color:var(--color-accent-link); color:var(--color-chip-text); background:var(--color-row-hover); }'
        ].join('\n');
        document.head.appendChild(style);
    }

    function parse(str) {
        str = (str || '').trim();
        var result = { attacks: 1, dCount: 1, dSides: 4, bonus: 0, critBuffIds: [] };
        if (!str) return result;

        if (str.indexOf('#') !== -1) {
            var hashIdx = str.indexOf('#');
            var critStr = str.substring(hashIdx + 1).trim();
            str = str.substring(0, hashIdx);
            if (critStr) {
                var parts = critStr.split(',');
                for (var i = 0; i < parts.length; i++) {
                    var id = parseInt(parts[i].trim(), 10);
                    if (id) result.critBuffIds.push(id);
                }
            }
        }

        if (str.indexOf('@') !== -1) {
            var atParts = str.split('@');
            result.attacks = parseInt(atParts[0], 10) || 1;
            str = atParts[1];
        }

        if (str.indexOf('+') !== -1) {
            var plusParts = str.split('+');
            str = plusParts[0];
            result.bonus = parseInt(plusParts[1], 10) || 0;
        } else {
            var dashIdx = str.indexOf('-', 1);
            if (dashIdx > 0) {
                result.bonus = -(parseInt(str.substring(dashIdx + 1), 10) || 0);
                str = str.substring(0, dashIdx);
            }
        }

        if (str.indexOf('d') !== -1) {
            var dParts = str.split('d');
            result.dCount = parseInt(dParts[0], 10) || 1;
            result.dSides = parseInt(dParts[1], 10) || 4;
        }

        return result;
    }

    function format(vals) {
        var s = '';
        if ((vals.attacks || 1) > 1) s += vals.attacks + '@';
        s += (vals.dCount || 1) + 'd' + (vals.dSides || 4);
        if (vals.bonus > 0) s += '+' + vals.bonus;
        else if (vals.bonus < 0) s += '-' + Math.abs(vals.bonus);
        if (vals.critBuffIds && vals.critBuffIds.length > 0) {
            s += '#' + vals.critBuffIds.join(',');
        }
        return s;
    }

    function avgDamage(vals) {
        var avg = vals.dCount * (vals.dSides + 1) / 2 + vals.bonus;
        var perAtk = Math.max(0, avg);
        var total = perAtk * (vals.attacks || 1);
        var min = Math.max(0, vals.dCount + vals.bonus) * (vals.attacks || 1);
        var max = (vals.dCount * vals.dSides + vals.bonus) * (vals.attacks || 1);
        return { perAtk: perAtk.toFixed(1), total: total.toFixed(1), min: min, max: max };
    }

    function attach(input) {
        injectStyles();

        var parent = input.parentElement;
        if (parent.classList.contains('drh-input-row')) return;

        var wrapper = document.createElement('div');
        wrapper.style.display = 'flex';
        wrapper.style.gap = '0.4rem';
        wrapper.style.alignItems = 'center';
        wrapper.className = 'drh-input-row';
        input.parentNode.insertBefore(wrapper, input);
        wrapper.appendChild(input);

        var btn = document.createElement('button');
        btn.type = 'button';
        btn.className = 'drh-toggle';
        btn.textContent = '⚙';
        btn.title = 'Dice roll builder';
        wrapper.appendChild(btn);

        var panel = document.createElement('div');
        panel.className = 'drh-panel';
        wrapper.parentNode.insertBefore(panel, wrapper.nextSibling);

        // --- Dice row ---
        var row = document.createElement('div');
        row.className = 'drh-row';

        var fields = {};

        row.appendChild(makeGroup('Attacks', 'number', 1, function(f) { fields.attacks = f; }));
        row.appendChild(makeSep('@'));
        row.appendChild(makeGroup('Dice', 'number', 1, function(f) { fields.dCount = f; }));
        row.appendChild(makeSep('d'));
        row.appendChild(makeGroup('Sides', 'number', 4, function(f) { fields.dSides = f; }));
        row.appendChild(makeSep('+'));
        row.appendChild(makeGroup('Bonus', 'number', 0, function(f) { fields.bonus = f; }));

        panel.appendChild(row);

        fields.bonus.min = -99;
        fields.attacks.min = 1;
        fields.dCount.min = 1;
        fields.dSides.min = 1;

        // --- Crit buff row ---
        var critRow = document.createElement('div');
        critRow.className = 'drh-crit-row';

        var critLabel = document.createElement('span');
        critLabel.className = 'drh-label';
        critLabel.textContent = 'Crit Buffs (#)';
        critRow.appendChild(critLabel);

        var chipContainer = document.createElement('span');
        chipContainer.className = 'drh-crit-chips';
        critRow.appendChild(chipContainer);

        var addBtn = document.createElement('button');
        addBtn.type = 'button';
        addBtn.className = 'drh-add-buff';
        addBtn.textContent = '+ Add';
        addBtn.title = 'Pick buff to apply on critical hit';
        critRow.appendChild(addBtn);

        panel.appendChild(critRow);

        // --- Preview ---
        var preview = document.createElement('div');
        preview.className = 'drh-preview';
        panel.appendChild(preview);

        // --- Crit buff chip management ---
        var critBuffIds = [];

        var renderChips = function() {
            chipContainer.innerHTML = '';
            for (var i = 0; i < critBuffIds.length; i++) {
                appendChip(critBuffIds[i]);
            }
        };

        var appendChip = function(id) {
            var chip = document.createElement('span');
            chip.className = 'drh-chip';
            chip.dataset.id = id;
            var label = escHtml(typeof PickerConfigs !== 'undefined' ? PickerConfigs.buffName(id) : '#' + id);
            chip.innerHTML = label + ' <button title="Remove">&times;</button>';
            chip.querySelector('button').addEventListener('click', function() {
                critBuffIds = critBuffIds.filter(function(v) { return v !== id; });
                renderChips();
                syncFromPanel();
            });
            chipContainer.appendChild(chip);
        };

        addBtn.addEventListener('click', function() {
            if (typeof Picker === 'undefined' || typeof PickerConfigs === 'undefined') return;
            Picker.open({
                title: 'Select Crit Buff',
                idKey: 'BuffId',
                columns: PickerConfigs.buffs.columns,
                searchKeys: PickerConfigs.buffs.searchKeys,
                sort: PickerConfigs.buffs.sort,
                source: PickerConfigs.buffs.source,
                multi: true,
                selected: critBuffIds.slice(),
                onSelect: function(buffs) {
                    critBuffIds = buffs.map(function(b) { return b.BuffId; });
                    renderChips();
                    syncFromPanel();
                }
            });
        });

        // --- Sync functions ---
        var syncFromPanel = function() {
            var vals = {
                attacks: parseInt(fields.attacks.value, 10) || 1,
                dCount: parseInt(fields.dCount.value, 10) || 1,
                dSides: parseInt(fields.dSides.value, 10) || 4,
                bonus: parseInt(fields.bonus.value, 10) || 0,
                critBuffIds: critBuffIds.slice()
            };
            input.value = format(vals);
            updatePreview(vals, preview);
        };

        var syncFromInput = function() {
            var vals = parse(input.value);
            fields.attacks.value = vals.attacks;
            fields.dCount.value = vals.dCount;
            fields.dSides.value = vals.dSides;
            fields.bonus.value = vals.bonus;
            critBuffIds = vals.critBuffIds.slice();
            renderChips();
            updatePreview(vals, preview);
        };

        // --- Wire events ---
        var fieldList = [fields.attacks, fields.dCount, fields.dSides, fields.bonus];
        for (var fi = 0; fi < fieldList.length; fi++) {
            fieldList[fi].addEventListener('input', syncFromPanel);
        }
        input.addEventListener('input', syncFromInput);
        input.addEventListener('change', syncFromInput);

        btn.addEventListener('click', function() {
            var isOpen = panel.classList.toggle('open');
            btn.classList.toggle('active', isOpen);
            if (isOpen) syncFromInput();
        });
    }

    function makeGroup(labelText, type, defaultVal, setter) {
        var group = document.createElement('div');
        group.className = 'drh-group';
        var label = document.createElement('label');
        label.textContent = labelText;
        group.appendChild(label);
        var inp = document.createElement('input');
        inp.type = type;
        inp.value = defaultVal;
        group.appendChild(inp);
        setter(inp);
        return group;
    }

    function makeSep(ch) {
        var sep = document.createElement('span');
        sep.className = 'drh-sep';
        sep.textContent = ch;
        return sep;
    }

    function updatePreview(vals, el) {
        var stats = avgDamage(vals);
        var atkLabel = (vals.attacks || 1) > 1 ? ' &times; ' + vals.attacks + ' attacks' : '';
        var critLabel = vals.critBuffIds && vals.critBuffIds.length > 0
            ? ' | crit buffs: #' + escHtml(vals.critBuffIds.join(', #'))
            : '';
        el.innerHTML = '<code>' + escHtml(format(vals)) + '</code>' +
            ' - <strong>' + stats.min + '&ndash;' + stats.max + ' dmg</strong>' +
            ' (avg ' + stats.total + ')' +
            atkLabel + critLabel;
    }

    function escHtml(s) {
        return String(s).replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;');
    }

    return { attach: attach, parse: parse, format: format };

})();
