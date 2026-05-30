/**
 * Convert a 5-field cron expression to a human-readable Vietnamese string.
 * Covers the common patterns used in payroll cron jobs.
 */
export function cronToVietnamese(expr: string): string {
  if (!expr) return expr;
  const parts = expr.trim().split(/\s+/);
  if (parts.length !== 5) return expr;

  const [min, hour, dom, month, dow] = parts;

  // Every minute
  if (min === '*' && hour === '*' && dom === '*' && month === '*' && dow === '*') {
    return 'Mỗi phút';
  }

  // Every hour at :MM
  if (hour === '*' && dom === '*' && month === '*' && dow === '*') {
    return `Mỗi giờ lúc :${min}`;
  }

  // Specific time, every day
  if (dom === '*' && month === '*' && dow === '*') {
    return `Hàng ngày lúc ${fmtTime(hour, min)}`;
  }

  // Specific day of week
  if (dow !== '*' && dom === '*' && month === '*') {
    const dayName = dayOfWeekName(dow);
    return `${dayName} hàng tuần lúc ${fmtTime(hour, min)}`;
  }

  // Specific day of month
  if (dom !== '*' && month === '*' && dow === '*') {
    return `Ngày ${dom} hàng tháng lúc ${fmtTime(hour, min)}`;
  }

  // Fallback
  return expr;
}

function fmtTime(hour: string, min: string): string {
  const h = hour === '*' ? 'mỗi giờ' : `${hour.padStart(2, '0')}:${min.padStart(2, '0')}`;
  return h;
}

function dayOfWeekName(dow: string): string {
  const names: Record<string, string> = {
    '0': 'Chủ nhật', '7': 'Chủ nhật',
    '1': 'Thứ Hai', '2': 'Thứ Ba', '3': 'Thứ Tư',
    '4': 'Thứ Năm', '5': 'Thứ Sáu', '6': 'Thứ Bảy',
  };
  if (names[dow]) return names[dow];
  // Handle comma-separated: "1,3,5"
  return dow.split(',').map(d => names[d] || d).join(', ');
}
