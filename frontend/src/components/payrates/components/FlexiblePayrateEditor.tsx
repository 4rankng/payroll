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
  const [bulkVal, setBulkVal]      = useState('');
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

  /* ── rates ── */
  const setRate = (pos: string, s: Shift, digits: string) => {
    const key = shiftKey(s.start, s.end);
    const num = parseInt(digits.replace(/\D/g,''),10)||0;
    const next = { ...rateMap, [pos]: { ...rateMap[pos], [key]: num } };
    setRateMap(next);
    emit(positions, shifts, next);
  };
  const getRate = useCallback((pos: string, s: Shift) => rateMap[pos]?.[shiftKey(s.start,s.end)] ?? 0, [rateMap]);

  /* ── quick fill ── */
  const fillEmpty = () => {
    const n = parseInt(bulkVal.replace(/\D/g,''),10);
    if (!n) return;
    const next = { ...rateMap };
    positions.forEach(pos => {
      next[pos] = { ...next[pos] };
      shifts.forEach(s => {
        const k = shiftKey(s.start,s.end);
        if (!next[pos][k]) next[pos][k] = n;
      });
    });
    setRateMap(next);
    emit(positions, shifts, next);
  };
  const clearAll = () => { setRateMap({}); emit(positions, shifts, {}); };
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
          0%   { box-shadow: 0 0 0 0 rgba(79,70,229,.5), 0 0 0 3px rgba(79,70,229,.15); border-color: #4f46e5; }
          65%  { box-shadow: 0 0 0 8px rgba(79,70,229,0), 0 0 0 3px rgba(79,70,229,.1); }
          100% { box-shadow: 0 0 0 0 rgba(79,70,229,0), 0 0 0 0 rgba(79,70,229,0); border-color: #e2e8f0; }
        }
        .shift-highlight { animation: shift-ping 1.1s cubic-bezier(.2,.7,.2,1) forwards; }
      `}</style>

      <div className="fpe flex flex-col gap-4">

        {/* ── Salary table card ── */}
        <div className="rounded-2xl border border-border bg-card overflow-hidden">

          {/* card header */}
          <div className="flex items-center justify-between px-6 py-4 border-b border-border/60">
            <span className="text-[11px] font-bold tracking-[.11em] text-muted-foreground uppercase">Bảng mức lương</span>
            {total > 0 && (
              <div className="flex items-center gap-2.5">
                <div className="w-28 h-1.5 rounded-full bg-muted overflow-hidden">
                  <div className="h-full rounded-full transition-all duration-300"
                    style={{ width:`${pct}%`, background: pct===100 ? '#059669' : '#4f46e5' }} />
                </div>
                <span className="fpe-mono text-xs font-semibold text-muted-foreground">{filled}/{total}</span>
              </div>
            )}
          </div>

          {/* quick-fill toolbar */}
          {showTable && !readOnly && (
            <div className="flex items-center gap-2.5 flex-wrap px-6 py-3 border-b border-border/40 bg-muted/20">
              <span className="text-xs font-semibold text-muted-foreground">Điền nhanh</span>
              <div className="relative">
                <input
                  className="fpe-mono w-32 h-8 pl-3 pr-6 rounded-lg border border-border bg-white text-sm text-right"
                  inputMode="numeric" placeholder="50.000"
                  value={bulkVal ? fmtVND(parseInt(bulkVal.replace(/\D/g,'')||'0',10)) : ''}
                  onChange={e => setBulkVal(e.target.value)}
                  onKeyDown={e => e.key==='Enter' && fillEmpty()}
                />
                <span className="absolute right-2 top-1/2 -translate-y-1/2 text-xs text-muted-foreground/60">₫</span>
              </div>
              <button onClick={fillEmpty} className="h-8 px-3 text-xs font-semibold rounded-lg border border-border bg-white hover:bg-muted/50 transition-colors">Điền ô trống</button>
              <button onClick={clearAll}  className="h-8 px-3 text-xs font-semibold rounded-lg border border-border bg-white text-muted-foreground hover:bg-muted/50 transition-colors">Xoá hết</button>
              <span className="text-xs text-muted-foreground/60 ml-auto hidden sm:block">Di chuột vào hàng → sao chép mức lương</span>
            </div>
          )}

          {/* table */}
          {showTable ? (
            <div className="overflow-auto" style={{ maxHeight: 440 }}>
              <table style={{ borderCollapse:'separate', borderSpacing:0, width:'max-content', minWidth:'100%' }}>
                <thead>
                  <tr>
                    {/* corner */}
                    <th style={{ position:'sticky', top:0, left:0, zIndex:4, background:'#fff', borderRight:'1px solid #e2e8f0', borderBottom:'1px solid #cbd5e1', padding:'12px 14px', width:176, minWidth:176, textAlign:'left' }}>
                      <div className="flex flex-col leading-tight">
                        <span className="text-[10px] text-muted-foreground/60 font-semibold">Ca làm việc →</span>
                        <span className="text-sm font-bold text-foreground">Vị trí ↓</span>
                      </div>
                    </th>
                    {/* shift columns */}
                    {shifts.map((s,i) => {
                      const ov = isON(s.start, s.end);
                      return (
                        <th key={i} style={{ position:'sticky', top:0, zIndex:3, background:'#fff', borderRight:'1px solid #e2e8f0', borderBottom:'1px solid #cbd5e1', padding:'10px 12px', width:152, minWidth:152, textAlign:'left', verticalAlign:'top' }}>
                          <div className="flex flex-col gap-0.5">
                            <div className="flex items-center justify-between gap-1.5">
                              <span className="fpe-mono text-sm font-bold tracking-tight">{s.start}–{s.end}</span>
                              {!readOnly && (
                                <button onClick={()=>removeShift(s)} className="w-5 h-5 flex items-center justify-center rounded text-muted-foreground hover:text-destructive hover:bg-destructive/10 transition-colors text-base leading-none">×</button>
                              )}
                            </div>
                            <div className="flex items-center gap-1.5">
                              {s.label && <span className="text-[11px] text-muted-foreground">{s.label}</span>}
                              <span className={`text-[11px] font-semibold ${ov?'text-purple-600':'text-muted-foreground/50'}`}>
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
                          className="flex items-center gap-1 px-2.5 py-1.5 rounded-lg border border-dashed border-border text-indigo-700 text-xs font-bold hover:border-indigo-400 hover:bg-indigo-50/50 transition-colors whitespace-nowrap">
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
                          <span className="text-sm font-bold">{pos}</span>
                          {!readOnly && (
                            <div className="flex gap-1 opacity-0 group-hover:opacity-100 transition-opacity">
                              <button onClick={()=>copyRow(pos)} title="Sao chép sang cả hàng" className="w-5 h-5 flex items-center justify-center rounded border border-border bg-white text-muted-foreground hover:border-indigo-400 hover:text-indigo-600 transition-colors text-xs">→</button>
                              <button onClick={()=>togglePos(pos)} title="Xoá vị trí" className="w-5 h-5 flex items-center justify-center rounded text-muted-foreground hover:text-destructive hover:bg-destructive/10 transition-colors text-base leading-none">×</button>
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
                                className="fpe-mono w-full h-9 pl-2 pr-7 rounded-lg text-sm text-right transition-all"
                                inputMode="numeric"
                                placeholder="0"
                                readOnly={readOnly}
                                value={focused ? (val||'') : (has ? fmtVND(val) : '')}
                                onFocus={() => setFocusKey(key)}
                                onBlur={() => setFocusKey(null)}
                                onChange={e => setRate(pos, s, e.target.value)}
                                style={{
                                  border: `1px solid ${focused ? '#4f46e5' : 'transparent'}`,
                                  boxShadow: focused ? '0 0 0 3px #eef1ff' : 'none',
                                  background: focused ? '#fff' : 'transparent',
                                  color: has ? '#047857' : '#0f172a',
                                  fontWeight: has ? 600 : 400,
                                }}
                              />
                              <span className="absolute right-2 top-1/2 -translate-y-1/2 text-[11px] pointer-events-none"
                                style={{ color: has ? '#bdeccf' : '#94a3b8' }}>₫</span>
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
              <div className="w-12 h-12 rounded-2xl bg-indigo-50 flex items-center justify-center">
                <svg viewBox="0 0 24 24" className="w-6 h-6 text-indigo-600" fill="none" stroke="currentColor" strokeWidth="2"><rect x="3" y="3" width="7" height="7" rx="1"/><rect x="14" y="3" width="7" height="7" rx="1"/><rect x="3" y="14" width="7" height="7" rx="1"/><rect x="14" y="14" width="7" height="7" rx="1"/></svg>
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
            <div className="grid grid-cols-1 sm:grid-cols-2 gap-4 p-5 border-t border-border/60">
              {/* positions */}
              <div className="rounded-xl border border-border p-4">
                <div className="flex items-center gap-2 mb-3">
                  <svg viewBox="0 0 24 24" className="w-3.5 h-3.5 text-indigo-600" fill="none" stroke="currentColor" strokeWidth="2"><path d="M16 19v-1a4 4 0 0 0-4-4H6a4 4 0 0 0-4 4v1"/><circle cx="9" cy="7" r="3"/><path d="M22 19v-1a4 4 0 0 0-3-3.87M16 4.13A4 4 0 0 1 16 11"/></svg>
                  <span className="text-xs font-bold tracking-[.08em] uppercase text-foreground/70">Vị trí</span>
                  <span className="ml-auto text-xs text-muted-foreground/60">chạm để thêm / bỏ</span>
                </div>
                <div className="flex flex-wrap gap-2">
                  {ALL_POSITIONS.map(name => (
                    <button key={name} onClick={()=>togglePos(name)}
                      className="fpe-sans inline-flex items-center gap-1 px-3 py-1.5 rounded-full text-xs font-semibold border transition-all"
                      style={{
                        border: `1px solid ${hasPos(name)?'#c7d0fc':'#e2e8f0'}`,
                        background: hasPos(name)?'#eef1ff':'#fff',
                        color: hasPos(name)?'#4338ca':'#334155',
                      }}>
                      {hasPos(name) && <svg viewBox="0 0 24 24" className="w-3 h-3" fill="none" stroke="currentColor" strokeWidth="2.5"><path d="M20 6 9 17l-5-5"/></svg>}
                      {name}
                    </button>
                  ))}
                </div>
              </div>

              {/* shifts */}
              <div className="rounded-xl border border-border p-4">
                <div className="flex items-center gap-2 mb-3">
                  <svg viewBox="0 0 24 24" className="w-3.5 h-3.5 text-indigo-600" fill="none" stroke="currentColor" strokeWidth="2"><circle cx="12" cy="12" r="9"/><path d="M12 7v5l3 2"/></svg>
                  <span className="text-xs font-bold tracking-[.08em] uppercase text-foreground/70">Ca làm việc</span>
                  <span className="ml-auto text-xs text-muted-foreground/60">mẫu phổ biến</span>
                </div>
                <div className="flex flex-wrap gap-2">
                  {PRESET_SHIFTS.map(p => {
                    const on = hasShift(p.start, p.end);
                    return (
                      <button key={p.start+p.end} onClick={()=>togglePreset(p)}
                        className="fpe-mono inline-flex items-center gap-1 px-3 py-1.5 rounded-full text-xs font-semibold border transition-all"
                        style={{
                          border:`1px solid ${on?'#c7d0fc':'#e2e8f0'}`,
                          background: on?'#eef1ff':'#fff',
                          color: on?'#4338ca':'#334155',
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
                  <div className="flex gap-2">
                    <input ref={shiftInputRef}
                      value={customShift}
                      onChange={e => { setCustomShift(e.target.value); setShiftErr(null); setShiftHighlight(false); }}
                      onKeyDown={e => e.key==='Enter' && addCustomShift()}
                      placeholder="06:00-14:00" inputMode="numeric"
                      className={`fpe-mono flex-1 h-9 px-3 rounded-lg text-sm border transition-colors ${shiftHighlight ? 'shift-highlight' : ''}`}
                      style={{
                        border: `1px solid ${shiftErr ? '#fbcfcf' : '#e2e8f0'}`,
                        background: shiftErr ? '#fef2f2' : '#fff',
                        color: '#0f172a',
                      }}
                    />
                    <button onClick={addCustomShift}
                      className="fpe-sans h-9 px-4 rounded-lg text-sm font-bold text-white transition-colors"
                      style={{ background:'#4f46e5' }}>
                      Thêm ca
                    </button>
                  </div>
                  {shiftErr && (
                    <div className="mt-1.5 text-xs text-red-600 flex items-center gap-1">
                      Sai định dạng — cần <span className="fpe-mono">HH:MM-HH:MM</span> (24 giờ)
                      {typeof shiftErr === 'object' && shiftErr.fix && (
                        <button
                          onClick={() => { setCustomShift(shiftErr.fix); setShiftErr(null); shiftInputRef.current?.focus(); }}
                          className="fpe-mono font-bold text-indigo-600 underline underline-offset-2 ml-1">
                          Dùng: {shiftErr.fix}
                        </button>
                      )}
                    </div>
                  )}
                </div>
              </div>
            </div>
          )}
        </div>

        {/* ── Footer stats bar ── */}
        <div className="rounded-2xl border border-border bg-card px-5 py-3.5 flex items-center gap-4 flex-wrap">
          <div className="flex items-center gap-3 text-sm text-muted-foreground flex-wrap">
            <span><b className="fpe-mono font-bold text-foreground">{positions.length}</b> vị trí</span>
            <span className="text-border">·</span>
            <span><b className="fpe-mono font-bold text-foreground">{shifts.length}</b> ca làm việc</span>
            <span className="text-border">·</span>
            <span><b className="fpe-mono font-bold text-foreground">{filled}</b> / {total} ô có mức lương</span>
          </div>
          <div className="ml-auto">
            <span className="inline-flex items-center gap-1.5 px-3 py-1 rounded-full text-xs font-bold border"
              style={{
                background: valid ? '#f0fdf6' : '#fff7ed',
                color:      valid ? '#047857' : '#b45309',
                border:     `1px solid ${valid ? '#bdeccf' : '#fed7aa'}`,
              }}>
              {valid
                ? <><svg viewBox="0 0 24 24" className="w-3 h-3" fill="none" stroke="currentColor" strokeWidth="2.5"><path d="M20 6 9 17l-5-5"/></svg> Sẵn sàng lưu</>
                : '⚠ Cần thêm mức lương'}
            </span>
          </div>
        </div>
      </div>
    </div>
  );
}
