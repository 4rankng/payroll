import { useState, useMemo, useRef, useCallback, useEffect } from 'react';
import type { PayrateStructure } from '../types';

/* ─── constants ─── */
const DAY_TYPE = 'ngày thường';
const SHIFT_RE = /^([01]\d|2[0-3]):[0-5]\d-([01]\d|2[0-3]):[0-5]\d$/;

const ALL_POSITIONS = [
  'Phổ Thông','Kinh Nghiệm','Học Việc','Thợ Điện','Thợ Cơ Khí',
  'Thợ Hàn','Công Nhân','Kỹ Thuật','Thủ Kho','Tổ Trưởng','Quản Đốc','Giám Sát',
];

const PRESET_SHIFTS = [
  { start:'06:00', end:'14:00', label:'Ca sáng' },
  { start:'08:00', end:'17:00', label:'Hành chính' },
  { start:'14:00', end:'22:00', label:'Ca chiều' },
  { start:'22:00', end:'06:00', label:'Ca đêm' },
  { start:'08:00', end:'20:00', label:'Ca 12h ngày' },
  { start:'20:00', end:'08:00', label:'Ca 12h đêm' },
];

/* ─── helpers ─── */
const fmtVND = (n: number) => new Intl.NumberFormat('vi-VN').format(n);
const toMin  = (t: string) => { const [h,m] = t.split(':').map(Number); return h*60+m; };
const isON   = (s: string, e: string) => toMin(e) <= toMin(s);
const durH   = (s: string, e: string) => { let d = toMin(e)-toMin(s); if (d<=0) d+=1440; return +(d/60).toFixed(d%60?1:0); };
const shiftKey = (s: string, e: string) => `${s}-${e}`;

function repairShift(raw: string): string | null {
  const parts = raw.trim().replace(/\s/g,'').split('-');
  if (parts.length !== 2) return null;
  const fix = (seg: string) => {
    const [h,m=''] = seg.split(':');
    const hh = parseInt(h,10), mm = parseInt(m||'0',10);
    if (isNaN(hh)||isNaN(mm)) return null;
    return `${String(Math.min(23,Math.max(0,hh))).padStart(2,'0')}:${String(Math.min(59,Math.max(0,mm))).padStart(2,'0')}`;
  };
  const a=fix(parts[0]), b=fix(parts[1]);
  if (!a||!b) return null;
  const out=`${a}-${b}`;
  return SHIFT_RE.test(out) ? out : null;
}

/* ─── parse PayrateStructure → internal state ─── */
interface Shift { start: string; end: string; label: string; }
type RateMap = Record<string, Record<string, number>>; // rates[pos][shiftKey]

function parseRates(rates: PayrateStructure): { positions: string[]; shifts: Shift[]; rateMap: RateMap } {
  const positions = Object.keys(rates);
  const shiftSet = new Map<string, Shift>();
  const rateMap: RateMap = {};

  positions.forEach(pos => {
    rateMap[pos] = {};
    const dayData = rates[pos]?.[DAY_TYPE] as Record<string, number> | undefined;
    if (!dayData) return;
    Object.entries(dayData).forEach(([key, val]) => {
      // Always preserve the rate value — even non-time-range keys like 'ca ngày'
      rateMap[pos][key] = typeof val === 'number' ? val : 0;
      if (!SHIFT_RE.test(key)) return; // Only create Shift objects for time-range keys
      if (!shiftSet.has(key)) {
        const match = key.match(/^(\d{2}:\d{2})-(\d{2}:\d{2})$/)!;
        const [, s, e] = match;
        const preset = PRESET_SHIFTS.find(p => shiftKey(p.start,p.end)===key);
        shiftSet.set(key, { start:s, end:e, label: preset?.label ?? (isON(s,e)?'Qua đêm':'') });
      }
    });
  });

  return { positions, shifts: Array.from(shiftSet.values()), rateMap };
}

/* ─── build PayrateStructure from internal state ─── */
function buildRates(positions: string[], shifts: Shift[], rateMap: RateMap): PayrateStructure {
  const result: PayrateStructure = {};
  positions.forEach(pos => {
    const dayData: Record<string, number> = {};
    shifts.forEach(s => {
      const key = shiftKey(s.start, s.end);
      dayData[key] = rateMap[pos]?.[key] ?? 0;
    });
    // Preserve non-shift keys (e.g. 'ca ngày') from rateMap to avoid silent data loss
    Object.entries(rateMap[pos] ?? {}).forEach(([key, val]) => {
      if (!(key in dayData)) dayData[key] = val;
    });
    result[pos] = { [DAY_TYPE]: dayData } as PayrateStructure[string];
  });
  return result;
}

/* ─── component ─── */
interface Props {
  rates: PayrateStructure;
  onChange: (rates: PayrateStructure) => void;
  readOnly?: boolean;
}

export function FlexiblePayrateEditor({ rates, onChange, readOnly = false }: Props) {
  const parsed = useMemo(() => parseRates(rates), [rates]);

  const [positions, setPositions] = useState<string[]>(
    parsed.positions.length ? parsed.positions : ['Phổ Thông']
  );
  const [shifts, setShifts]       = useState<Shift[]>(parsed.shifts);
  const [rateMap, setRateMap]      = useState<RateMap>(parsed.rateMap);
  const [focusKey, setFocusKey]    = useState<string|null>(null);
  // Inline editing states for shift headers and position names
  const [editingShiftIdx, setEditingShiftIdx] = useState<number|null>(null);
  const [editingShiftValue, setEditingShiftValue] = useState('');
  const [editingPositionName, setEditingPositionName] = useState<string|null>(null);
  const [editingPositionValue, setEditingPositionValue] = useState('');

  // Track emitted rates to avoid re-syncing from our own updates
  const lastEmittedRef = useRef<PayrateStructure>(rates);

  // Re-sync internal state when parent rates change externally
  // (e.g. initial load, server data arriving, undo/redo)
  useEffect(() => {
    if (lastEmittedRef.current === rates) return;
    const p = parseRates(rates);
    setPositions(p.positions.length ? p.positions : ['Phổ Thông']);
    setShifts(p.shifts);
    setRateMap(p.rateMap);
  }, [rates]);
  const [customShift, setCustomShift] = useState('');
  const [shiftErr, setShiftErr]    = useState<{fix:string}|true|null>(null);
  const [shiftHighlight, setShiftHighlight] = useState(false);
  const shiftInputRef = useRef<HTMLInputElement>(null);

  /* ── emit changes ── */
  const emit = useCallback((p: string[], s: Shift[], r: RateMap) => {
    const built = buildRates(p, s, r);
    lastEmittedRef.current = built;
    onChange(built);
  }, [onChange]);

  /* ── positions ── */
  const hasPos = (name: string) => positions.includes(name);
  const togglePos = (name: string) => {
    const next = hasPos(name) ? positions.filter(p=>p!==name) : [...positions, name];
    setPositions(next);
    emit(next, shifts, rateMap);
  };

  /* ── shifts ── */
  const hasShift = (s: string, e: string) => shifts.some(x=>x.start===s&&x.end===e);
  const togglePreset = (p: typeof PRESET_SHIFTS[0]) => {
    const next = hasShift(p.start, p.end)
      ? shifts.filter(x=>!(x.start===p.start&&x.end===p.end))
      : [...shifts, p];
    setShifts(next);
    emit(positions, next, rateMap);
  };
  const removeShift = (s: Shift) => {
    const next = shifts.filter(x=>!(x.start===s.start&&x.end===s.end));
    setShifts(next);
    emit(positions, next, rateMap);
  };
  const addCustomShift = () => {
    const v = customShift.trim();
    if (SHIFT_RE.test(v)) {
      const [s,e] = v.split('-');
      if (!hasShift(s,e)) {
        const next = [...shifts, { start:s, end:e, label: isON(s,e)?'Qua đêm':'' }];
        setShifts(next);
        emit(positions, next, rateMap);
      }
      setCustomShift(''); setShiftErr(null);
      return;
    }
    const fixed = repairShift(v);
    setShiftErr(fixed ? {fix:fixed} : true);
  };

  /* ── inline shift header editing ── */
  const startShiftEdit = (idx: number) => {
    if (readOnly) return;
    const s = shifts[idx];
    setEditingShiftIdx(idx);
    setEditingShiftValue(`${s.start}-${s.end}`);
  };
  const cancelShiftEdit = () => { setEditingShiftIdx(null); setEditingShiftValue(''); };
  const saveShiftEdit = () => {
    if (editingShiftIdx === null) return;
    const val = editingShiftValue.trim();
    const fixed = repairShift(val) || (SHIFT_RE.test(val) ? val : null);
    if (!fixed) { cancelShiftEdit(); return; }
    const [newStart, newEnd] = fixed.split('-');
    const oldShift = shifts[editingShiftIdx];
    const oldKey = shiftKey(oldShift.start, oldShift.end);
    const newKey = shiftKey(newStart, newEnd);
    if (oldKey === newKey) { cancelShiftEdit(); return; }
    if (shifts.some((s, i) => i !== editingShiftIdx && shiftKey(s.start, s.end) === newKey)) { cancelShiftEdit(); return; }
    const preset = PRESET_SHIFTS.find(p => shiftKey(p.start, p.end) === newKey);
    const newShift: Shift = { start: newStart, end: newEnd, label: preset?.label ?? (isON(newStart, newEnd) ? 'Qua đêm' : '') };
    const newShifts = shifts.map((s, i) => i === editingShiftIdx ? newShift : s);
    const newRateMap = { ...rateMap };
    positions.forEach(pos => {
      if (newRateMap[pos] && oldKey in newRateMap[pos]) {
        const v = newRateMap[pos][oldKey];
        const updated = { ...newRateMap[pos] };
        delete updated[oldKey];
        updated[newKey] = v;
        newRateMap[pos] = updated;
      }
    });
    setShifts(newShifts);
    setRateMap(newRateMap);
    emit(positions, newShifts, newRateMap);
    cancelShiftEdit();
  };

  /* ── inline position name editing ── */
  const startPositionEdit = (pos: string) => {
    if (readOnly) return;
    setEditingPositionName(pos);
    setEditingPositionValue(pos);
  };
  const cancelPositionEdit = () => { setEditingPositionName(null); setEditingPositionValue(''); };
  const savePositionEdit = () => {
    const newName = editingPositionValue.trim();
    const oldName = editingPositionName;
    if (!newName || !oldName || newName === oldName) { cancelPositionEdit(); return; }
    if (positions.includes(newName)) { cancelPositionEdit(); return; }
    const newPositions = positions.map(p => p === oldName ? newName : p);
    const newRateMap: RateMap = {};
    positions.forEach(p => {
      const key = p === oldName ? newName : p;
      newRateMap[key] = rateMap[p] ?? {};
    });
    setPositions(newPositions);
    setRateMap(newRateMap);
    emit(newPositions, shifts, newRateMap);
    cancelPositionEdit();
  };

  /* ── rates ── */
  const setRate = (pos: string, s: Shift, digits: string) => {
    const key = shiftKey(s.start, s.end);
    const num = parseInt(digits.replace(/\D/g,''),10)||0;
    const next = { ...rateMap, [pos]: { ...rateMap[pos], [key]: num } };
    setRateMap(next);
    emit(positions, shifts, next);
  };
  const getRate = useCallback((pos: string, s: Shift) => rateMap[pos]?.[shiftKey(s.start,s.end)] ?? 0, [rateMap]);

  /* ── row copy ── */
  const copyRow = (pos: string) => {
    const first = shifts.map(s=>getRate(pos,s)).find(v=>v>0);
    if (!first) return;
    const next = { ...rateMap, [pos]: {} };
    shifts.forEach(s => { next[pos][shiftKey(s.start,s.end)] = first; });
    setRateMap(next);
    emit(positions, shifts, next);
  };

  /* ── stats ── */
  const total  = positions.length * shifts.length;
  const filled = useMemo(() => positions.reduce((acc,pos) =>
    acc + shifts.filter(s=>getRate(pos,s)>0).length, 0
  ), [positions, shifts, getRate]);
  const pct    = total ? Math.round(filled/total*100) : 0;
  const valid  = positions.length>0 && shifts.length>0 && filled>0;
  const showTable = positions.length>0 && shifts.length>0;

  return (
    <div className="flex flex-col gap-4">
      <style>{`
        @import url('https://fonts.googleapis.com/css2?family=Plus+Jakarta+Sans:wght@400;500;600;700;800&family=JetBrains+Mono:wght@400;500;600;700&display=swap');
        .fpe-mono { font-family: 'JetBrains Mono', ui-monospace, monospace; }
        .fpe-sans { font-family: 'Plus Jakarta Sans', system-ui, sans-serif; }
        .fpe input:focus { outline: none; }
        .fpe *::-webkit-scrollbar { height:8px; width:8px; }
        .fpe *::-webkit-scrollbar-thumb { background:#cbd5e1; border-radius:6px; border:2px solid #fff; }
        @keyframes shift-ping {
          0%   { box-shadow: 0 0 0 0 rgba(8,120,62,.5), 0 0 0 3px rgba(8,120,62,.15); border-color: #08783e; }
          65%  { box-shadow: 0 0 0 8px rgba(8,120,62,0), 0 0 0 3px rgba(8,120,62,.1); }
          100% { box-shadow: 0 0 0 0 rgba(8,120,62,0), 0 0 0 0 rgba(8,120,62,0); border-color: #e2e8f0; }
        }
        .shift-highlight { animation: shift-ping 1.1s cubic-bezier(.2,.7,.2,1) forwards; }
      `}</style>

      <div className="fpe flex flex-col gap-4">

        {/* ── Salary table card ── */}
        <div className="flex flex-col gap-4">

          {/* table */}
          {showTable ? (
            <div className="overflow-auto" style={{ maxHeight: 440 }}>
              <table style={{ borderCollapse:'separate', borderSpacing:0, width:'max-content', minWidth:'100%' }}>
                <thead>
                  <tr>
                    {/* corner */}
                    <th style={{ position:'sticky', top:0, left:0, zIndex:4, background:'#fff', borderRight:'1px solid #e2e8f0', borderBottom:'1px solid #cbd5e1', padding:'12px 14px', width:176, minWidth:176, textAlign:'left' }}>
                      <div className="flex flex-col leading-tight">
                        <span className="text-xs text-muted-foreground/60 font-semibold">Ca làm việc →</span>
                        <span className="text-sm font-bold text-foreground">Vị trí ↓</span>
                        <span className="mt-1 text-xs font-semibold text-emerald-700">Lương trọn ca (₫)</span>
                      </div>
                    </th>
                    {/* shift columns */}
                    {shifts.map((s,i) => {
                      const ov = isON(s.start, s.end);
                      return (
                        <th key={i} style={{ position:'sticky', top:0, zIndex:3, background:'#fff', borderRight:'1px solid #e2e8f0', borderBottom:'1px solid #cbd5e1', padding:'10px 12px', width:152, minWidth:152, textAlign:'left', verticalAlign:'top' }}>
                          <div className="flex flex-col gap-0.5">
                            <div className="flex items-center justify-between gap-1.5">
                              {editingShiftIdx === i ? (
                                <input
                                  className="fpe-mono h-11 w-[120px] rounded border border-emerald-400 bg-white px-2 text-sm font-bold tracking-tight focus:outline-none focus:ring-1 focus:ring-emerald-200"
                                  value={editingShiftValue}
                                  onChange={e => setEditingShiftValue(e.target.value)}
                                  onKeyDown={e => {
                                    if (e.key === 'Enter') { e.stopPropagation(); saveShiftEdit(); }
                                    if (e.key === 'Escape') { e.stopPropagation(); cancelShiftEdit(); }
                                  }}
                                  onBlur={() => saveShiftEdit()}
                                  autoFocus
                                />
                              ) : readOnly ? (
                                <span className="fpe-mono inline-flex min-h-11 items-center text-sm font-bold tracking-tight">
                                  {s.start}–{s.end}
                                </span>
                              ) : (
                                <button
                                  type="button"
                                  className="fpe-mono min-h-11 cursor-pointer text-left text-sm font-bold tracking-tight transition-colors hover:text-emerald-700"
                                  onClick={() => startShiftEdit(i)}
                                  title="Nhấn để đổi thời gian ca"
                                  aria-label={`Chỉnh khung giờ ${s.start}-${s.end}`}
                                >
                                  {s.start}–{s.end}
                                </button>
                              )}
                              {!readOnly && (
                                <button onClick={()=>removeShift(s)} className="flex h-11 w-11 items-center justify-center rounded text-base leading-none text-muted-foreground transition-colors hover:bg-destructive/10 hover:text-destructive">×</button>
                              )}
                            </div>
                            <div className="flex items-center gap-1.5">
                              {s.label && <span className="text-xs text-muted-foreground">{s.label}</span>}
                              <span className={`text-xs font-semibold ${ov?'text-emerald-700':'text-muted-foreground/50'}`}>
                                {ov && '🌙'}{durH(s.start,s.end)}h
                              </span>
                            </div>
                          </div>
                        </th>
                      );
                    })}
                    {/* add-shift column header */}
                    {!readOnly && (
                      <th style={{ position:'sticky', top:0, zIndex:3, background:'#fff', borderBottom:'1px solid #cbd5e1', padding:'10px 12px', width:120, minWidth:120 }}>
                        <button
                          onClick={() => {
                            shiftInputRef.current?.focus();
                            shiftInputRef.current?.scrollIntoView({ behavior: 'smooth', block: 'center' });
                            setShiftHighlight(true);
                            setTimeout(() => setShiftHighlight(false), 1200);
                          }}
                          className="flex min-h-11 items-center gap-1 rounded-lg border border-dashed border-border px-3 text-xs font-bold text-emerald-700 transition-colors hover:border-emerald-400 hover:bg-emerald-50/50">
                          + Thêm ca
                        </button>
                      </th>
                    )}
                  </tr>
                </thead>
                <tbody>
                  {positions.map((pos, ri) => (
                    <tr key={pos} style={{ background: ri%2 ? '#fafbfe' : '#fff' }} className="group">
                      {/* position cell */}
                      <th style={{ position:'sticky', left:0, zIndex:2, borderRight:'1px solid #cbd5e1', borderBottom:'1px solid #e2e8f0', padding:'8px 14px', textAlign:'left', background: ri%2?'#fafbfe':'#fff' }}>
                        <div className="flex items-center justify-between gap-2">
                          {editingPositionName === pos ? (
                            <input
                              className="h-11 w-full max-w-[140px] rounded border border-emerald-400 bg-white px-2 text-sm font-bold focus:outline-none focus:ring-1 focus:ring-emerald-200"
                              value={editingPositionValue}
                              onChange={e => setEditingPositionValue(e.target.value)}
                              onKeyDown={e => {
                                if (e.key === 'Enter') { e.stopPropagation(); savePositionEdit(); }
                                if (e.key === 'Escape') { e.stopPropagation(); cancelPositionEdit(); }
                              }}
                              onBlur={() => savePositionEdit()}
                              autoFocus
                            />
                          ) : (
                            <span
                              className={`text-sm font-bold ${!readOnly ? 'cursor-pointer hover:text-emerald-700 transition-colors' : ''}`}
                              onClick={() => startPositionEdit(pos)}
                              title={readOnly ? undefined : 'Nhấn để đổi tên vị trí'}
                            >
                              {pos}
                            </span>
                          )}
                          {!readOnly && (
                            <div className="flex gap-1 opacity-0 group-hover:opacity-100 transition-opacity">
                              <button onClick={()=>copyRow(pos)} title="Sao chép sang cả hàng" className="flex h-11 w-11 items-center justify-center rounded border border-border bg-white text-xs text-muted-foreground transition-colors hover:border-emerald-400 hover:text-emerald-700">→</button>
                              <button onClick={()=>togglePos(pos)} title="Xoá vị trí" className="flex h-11 w-11 items-center justify-center rounded text-base leading-none text-muted-foreground transition-colors hover:bg-destructive/10 hover:text-destructive">×</button>
                            </div>
                          )}
                        </div>
                      </th>
                      {/* rate cells */}
                      {shifts.map((s, ci) => {
                        const key = `${pos}__${ci}`;
                        const val = getRate(pos, s);
                        const has = val > 0;
                        const focused = focusKey === key;
                        return (
                          <td key={ci} style={{ borderRight:'1px solid #e2e8f0', borderBottom:'1px solid #e2e8f0', padding:4, background: has ? '#f0fdf6' : 'transparent' }}>
                            <div className="relative">
                              <input
                                className="fpe-mono h-11 w-full rounded-lg pl-2 pr-12 text-right text-sm transition-all"
                                inputMode="numeric"
                                placeholder="0"
                                aria-label={`Lương trọn ca cho ${pos}, ca ${s.start}-${s.end}`}
                                readOnly={readOnly}
                                value={focused ? (val||'') : (has ? fmtVND(val) : '')}
                                onFocus={() => setFocusKey(key)}
                                onBlur={() => setFocusKey(null)}
                                onChange={e => setRate(pos, s, e.target.value)}
                                style={{
                                  border: `1px solid ${focused ? '#08783e' : 'transparent'}`,
                                  boxShadow: focused ? '0 0 0 3px #e6f5ed' : 'none',
                                  background: focused ? '#fff' : 'transparent',
                                  color: has ? '#047857' : '#0f172a',
                                  fontWeight: has ? 600 : 400,
                                }}
                              />
                              <span className="pointer-events-none absolute right-2 top-1/2 -translate-y-1/2 text-xs"
                                style={{ color: has ? '#047857' : '#94a3b8' }}>₫/ca</span>
                            </div>
                          </td>
                        );
                      })}
                      {!readOnly && <td style={{ borderBottom:'1px solid #e2e8f0', background:'transparent' }} />}
                    </tr>
                  ))}
                  {/* hint row */}
                  {!readOnly && (
                    <tr>
                      <th style={{ position:'sticky', left:0, zIndex:2, background:'#fff', borderRight:'1px solid #cbd5e1', padding:'8px 14px', textAlign:'left' }}>
                        <span className="text-xs text-muted-foreground/50">↓ chọn ở dưới</span>
                      </th>
                      <td colSpan={shifts.length+1} style={{ padding:'8px 12px', background:'#fff' }}>
                        <span className="text-xs text-muted-foreground/50">Thêm vị trí từ danh sách bên dưới</span>
                      </td>
                    </tr>
                  )}
                </tbody>
              </table>
            </div>
          ) : (
            /* empty state */
            <div className="flex flex-col items-center justify-center py-12 gap-3 border-t border-border">
              <div className="w-12 h-12 rounded-2xl bg-emerald-50 flex items-center justify-center">
                <svg viewBox="0 0 24 24" className="w-6 h-6 text-emerald-700" fill="none" stroke="currentColor" strokeWidth="2"><rect x="3" y="3" width="7" height="7" rx="1"/><rect x="14" y="3" width="7" height="7" rx="1"/><rect x="3" y="14" width="7" height="7" rx="1"/><rect x="14" y="14" width="7" height="7" rx="1"/></svg>
              </div>
              <p className="text-sm font-bold text-foreground">Ma trận đang trống</p>
              <p className="text-xs text-muted-foreground text-center max-w-xs">
                {positions.length===0&&shifts.length===0 ? 'Thêm ít nhất một vị trí và một ca làm việc'
                  : positions.length===0 ? 'Thêm ít nhất một vị trí'
                  : 'Thêm ít nhất một ca làm việc'} ở danh sách bên dưới để bắt đầu nhập mức lương.
              </p>
            </div>
          )}

          {/* ── palettes ── */}
          {!readOnly && (
            <div
              className="grid grid-cols-1 divide-y divide-border/60 border-t border-border/60 sm:grid-cols-2 sm:divide-x sm:divide-y-0"
              data-slot="payrate-selector-sections"
            >
              {/* positions */}
              <section
                className="min-w-0 p-4 sm:p-5 sm:pr-6"
                aria-labelledby="payrate-positions-heading"
              >
                <div className="mb-3 flex flex-wrap items-center gap-x-2 gap-y-1">
                  <svg viewBox="0 0 24 24" className="w-3.5 h-3.5 text-emerald-700" fill="none" stroke="currentColor" strokeWidth="2"><path d="M16 19v-1a4 4 0 0 0-4-4H6a4 4 0 0 0-4 4v1"/><circle cx="9" cy="7" r="3"/><path d="M22 19v-1a4 4 0 0 0-3-3.87M16 4.13A4 4 0 0 1 16 11"/></svg>
                  <h3 id="payrate-positions-heading" className="text-xs font-bold tracking-[.08em] uppercase text-foreground/70">Vị trí</h3>
                  <span className="w-full pl-[22px] text-xs text-muted-foreground/70 sm:ml-auto sm:w-auto sm:pl-0">Chọn để thêm hoặc bỏ</span>
                </div>
                <div className="flex flex-wrap gap-2">
                  {ALL_POSITIONS.map(name => (
                    <button
                      key={name}
                      type="button"
                      aria-pressed={hasPos(name)}
                      onClick={()=>togglePos(name)}
                      className="fpe-sans inline-flex min-h-11 items-center gap-1 rounded-full border px-3 text-xs font-semibold transition-all"
                      style={{
                        border: `1px solid ${hasPos(name)?'#bfe0cc':'#e2e8f0'}`,
                        background: hasPos(name)?'#e8f3ec':'#fff',
                        color: hasPos(name)?'#08783e':'#334155',
                      }}>
                      {hasPos(name) && <svg viewBox="0 0 24 24" className="w-3 h-3" fill="none" stroke="currentColor" strokeWidth="2.5"><path d="M20 6 9 17l-5-5"/></svg>}
                      {name}
                    </button>
                  ))}
                </div>
              </section>

              {/* shifts */}
              <section
                className="min-w-0 p-4 sm:p-5 sm:pl-6"
                aria-labelledby="payrate-shifts-heading"
              >
                <div className="mb-3 flex flex-wrap items-center gap-x-2 gap-y-1">
                  <svg viewBox="0 0 24 24" className="w-3.5 h-3.5 text-emerald-700" fill="none" stroke="currentColor" strokeWidth="2"><circle cx="12" cy="12" r="9"/><path d="M12 7v5l3 2"/></svg>
                  <h3 id="payrate-shifts-heading" className="text-xs font-bold tracking-[.08em] uppercase text-foreground/70">Ca làm việc</h3>
                  <span className="w-full pl-[22px] text-xs text-muted-foreground/70 sm:ml-auto sm:w-auto sm:pl-0">Chọn ca phổ biến</span>
                </div>
                <div className="flex flex-wrap gap-2">
                  {PRESET_SHIFTS.map(p => {
                    const on = hasShift(p.start, p.end);
                    return (
                      <button
                        key={p.start+p.end}
                        type="button"
                        aria-pressed={on}
                        onClick={()=>togglePreset(p)}
                        className="fpe-mono inline-flex min-h-11 items-center gap-1 rounded-full border px-3 text-xs font-semibold transition-all"
                        style={{
                          border:`1px solid ${on?'#bfe0cc':'#e2e8f0'}`,
                          background: on?'#e8f3ec':'#fff',
                          color: on?'#08783e':'#334155',
                        }}>
                        {on && <svg viewBox="0 0 24 24" className="w-3 h-3" fill="none" stroke="currentColor" strokeWidth="2.5"><path d="M20 6 9 17l-5-5"/></svg>}
                        {p.start}–{p.end}
                        <span className="fpe-sans opacity-60 font-medium ml-1">{p.label}</span>
                      </button>
                    );
                  })}
                </div>

                {/* custom shift input */}
                <div className="mt-3.5 pt-3.5 border-t border-dashed border-border/70">
                  <div className="grid grid-cols-1 gap-2 min-[380px]:grid-cols-[minmax(0,1fr)_auto]">
                    <input ref={shiftInputRef}
                      value={customShift}
                      onChange={e => { setCustomShift(e.target.value); setShiftErr(null); setShiftHighlight(false); }}
                      onKeyDown={e => e.key==='Enter' && addCustomShift()}
                      placeholder="06:00-14:00" inputMode="numeric"
                      aria-label="Khung giờ ca làm việc mới"
                      className={`fpe-mono h-11 rounded-lg border px-3 text-sm transition-colors ${shiftHighlight ? 'shift-highlight' : ''}`}
                      style={{
                        border: `1px solid ${shiftErr ? '#fbcfcf' : '#e2e8f0'}`,
                        background: shiftErr ? '#fef2f2' : '#fff',
                        color: '#0f172a',
                      }}
                    />
                    <button
                      type="button"
                      onClick={addCustomShift}
                      className="fpe-sans h-11 rounded-lg px-4 text-sm font-bold text-white transition-colors"
                      style={{ background:'#08783e' }}>
                      Thêm ca
                    </button>
                  </div>
                  {shiftErr && (
                    <div className="mt-1.5 text-xs text-red-600 flex items-center gap-1">
                      Sai định dạng — cần <span className="fpe-mono">HH:MM-HH:MM</span> (24 giờ)
                      {typeof shiftErr === 'object' && shiftErr.fix && (
                        <button
                          type="button"
                          onClick={() => { setCustomShift(shiftErr.fix); setShiftErr(null); shiftInputRef.current?.focus(); }}
                          className="fpe-mono font-bold text-emerald-700 underline underline-offset-2 ml-1">
                          Dùng: {shiftErr.fix}
                        </button>
                      )}
                    </div>
                  )}
                </div>
              </section>
            </div>
          )}
        </div>

      </div>
    </div>
  );
}
