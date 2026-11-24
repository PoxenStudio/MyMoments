const WEEKDAYS = ['周日', '周一', '周二', '周三', '周四', '周五', '周六'];
const WEEKDAYS_SHORT = ['日', '一', '二', '三', '四', '五', '六'];

export function pad2(n: number): string {
  return n < 10 ? `0${n}` : String(n);
}

export function toISODate(date: Date): string {
  return `${date.getFullYear()}-${pad2(date.getMonth() + 1)}-${pad2(date.getDate())}`;
}

export function todayISO(): string {
  return toISODate(new Date());
}

export function parseISODate(value: string): Date {
  const [y, m, d] = value.split('-').map(Number);
  return new Date(y, (m ?? 1) - 1, d ?? 1, 12, 0, 0, 0);
}

export function isValidISODate(value: string | null | undefined): value is string {
  return !!value && /^\d{4}-\d{2}-\d{2}$/.test(value);
}

export function addDays(value: string, days: number): string {
  const d = parseISODate(value);
  d.setDate(d.getDate() + days);
  return toISODate(d);
}

export function addMonths(value: string, months: number): string {
  const d = parseISODate(value);
  d.setMonth(d.getMonth() + months);
  return toISODate(d);
}

export function startOfWeekISO(value: string, weekStart = 1): string {
  const d = parseISODate(value);
  const day = d.getDay();
  const diff = (day - weekStart + 7) % 7;
  d.setDate(d.getDate() - diff);
  return toISODate(d);
}

export function formatDateCN(value: string | null | undefined, opts: { withYear?: boolean; weekday?: boolean } = {}): string {
  if (!isValidISODate(value)) return '';
  const d = parseISODate(value);
  const parts = opts.withYear === false ? `${d.getMonth() + 1}月${d.getDate()}日` : `${d.getFullYear()}年${d.getMonth() + 1}月${d.getDate()}日`;
  return opts.weekday ? `${parts} ${WEEKDAYS[d.getDay()]}` : parts;
}

export function formatShortDate(value: string | null | undefined): string {
  if (!isValidISODate(value)) return '';
  const d = parseISODate(value);
  return `${d.getMonth() + 1}月${d.getDate()}日`;
}

export function formatMonthCN(value: string): string {
  const d = parseISODate(value);
  return `${d.getFullYear()}年${d.getMonth() + 1}月`;
}

export function weekdayCN(value: string): string {
  return WEEKDAYS[parseISODate(value).getDay()];
}

export function weekdayShortCN(index: number): string {
  return WEEKDAYS_SHORT[index] ?? '';
}

export function formatTimestamp(ts: number | null | undefined, opts: { dateOnly?: boolean; withYear?: boolean } = {}): string {
  if (!ts) return '';
  const d = new Date(ts * 1000);
  const date = `${d.getFullYear()}-${pad2(d.getMonth() + 1)}-${pad2(d.getDate())}`;
  if (opts.dateOnly) return formatDateCN(date, { withYear: opts.withYear });
  return `${formatDateCN(date, { withYear: opts.withYear })} ${pad2(d.getHours())}:${pad2(d.getMinutes())}`;
}

export function formatLocalDateTime(value: string | null | undefined): string {
  if (!value) return '';
  const [date, time] = value.split(' ');
  const trimmed = (time ?? '').slice(0, 5);
  const dateText = formatDateCN(date, { withYear: new Date().getFullYear() !== Number(date.slice(0, 4)) });
  return trimmed ? `${dateText} ${trimmed}` : dateText;
}

export function formatTime(value: string | null | undefined): string {
  if (!value) return '';
  return value.slice(0, 5);
}

export function toLocalDateTime(date: string, time: string): string {
  return `${date} ${time.length === 5 ? `${time}:00` : time}`;
}

export function relativeDayLabel(value: string): string {
  const today = todayISO();
  if (value === today) return '今天';
  if (value === addDays(today, 1)) return '明天';
  if (value === addDays(today, -1)) return '昨天';
  const diff = Math.round((parseISODate(value).getTime() - parseISODate(today).getTime()) / 86400000);
  if (diff > 1 && diff <= 7) return `${diff} 天后`;
  if (diff < -1 && diff >= -7) return `${-diff} 天前`;
  return formatDateCN(value);
}

export interface MonthCell {
  date: string;
  day: number;
  inMonth: boolean;
  isToday: boolean;
}

export function monthMatrix(year: number, month: number, weekStart = 1): MonthCell[] {
  const first = new Date(year, month - 1, 1, 12);
  const startOffset = (first.getDay() - weekStart + 7) % 7;
  const start = new Date(year, month - 1, 1 - startOffset, 12);
  const cells: MonthCell[] = [];
  const today = todayISO();
  for (let i = 0; i < 42; i += 1) {
    const d = new Date(start);
    d.setDate(start.getDate() + i);
    const iso = toISODate(d);
    cells.push({ date: iso, day: d.getDate(), inMonth: d.getMonth() === month - 1, isToday: iso === today });
  }
  return cells;
}

export function daysUntilBirthday(month: number, day: number): number {
  const now = new Date();
  const today = new Date(now.getFullYear(), now.getMonth(), now.getDate(), 12);
  let target = new Date(now.getFullYear(), month - 1, day, 12);
  if (month === 2 && day === 29 && target.getMonth() !== 1) target = new Date(now.getFullYear(), 1, 28, 12);
  if (target < today) {
    let next = new Date(now.getFullYear() + 1, month - 1, day, 12);
    if (month === 2 && day === 29 && next.getMonth() !== 1) next = new Date(now.getFullYear() + 1, 1, 28, 12);
    target = next;
  }
  return Math.round((target.getTime() - today.getTime()) / 86400000);
}

export function ageOnBirthday(birthYear: number, month: number, day: number): number {
  const now = new Date();
  let age = now.getFullYear() - birthYear;
  const target = new Date(now.getFullYear(), month - 1, day);
  if (target > now) age -= 1;
  return age;
}

export function greeting(name: string): string {
  const hour = new Date().getHours();
  if (hour < 6) return `夜深了，${name}`;
  if (hour < 11) return `早上好，${name}`;
  if (hour < 14) return `中午好，${name}`;
  if (hour < 18) return `下午好，${name}`;
  return `晚上好，${name}`;
}

export function monthRange(year: number, month: number): { start: string; end: string } {
  const start = `${year}-${pad2(month)}-01`;
  const last = new Date(year, month, 0, 12);
  return { start, end: toISODate(last) };
}
