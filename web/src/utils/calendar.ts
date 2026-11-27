export const WEEKDAYS = ['一', '二', '三', '四', '五', '六', '日'];

const LUNAR_DAYS = [
  '', '初一', '初二', '初三', '初四', '初五', '初六', '初七', '初八', '初九', '初十',
  '十一', '十二', '十三', '十四', '十五', '十六', '十七', '十八', '十九', '二十',
  '廿一', '廿二', '廿三', '廿四', '廿五', '廿六', '廿七', '廿八', '廿九', '三十',
];

export function lunarDayLabel(day: number): string {
  return LUNAR_DAYS[day] ?? '';
}

export function parseDate(value: string): Date {
  const [y, m, d] = value.slice(0, 10).split('-').map(Number);
  return new Date(y, (m || 1) - 1, d || 1);
}

export function toISODate(date: Date): string {
  const y = date.getFullYear();
  const m = String(date.getMonth() + 1).padStart(2, '0');
  const d = String(date.getDate()).padStart(2, '0');
  return `${y}-${m}-${d}`;
}

export function addDays(value: string, days: number): string {
  const date = parseDate(value);
  date.setDate(date.getDate() + days);
  return toISODate(date);
}

export function addMonths(value: string, months: number): string {
  const date = parseDate(value);
  date.setMonth(date.getMonth() + months);
  return toISODate(date);
}

export function startOfWeek(value: string, weekStart = 1): string {
  const date = parseDate(value);
  const diff = (date.getDay() - weekStart + 7) % 7;
  date.setDate(date.getDate() - diff);
  return toISODate(date);
}

export function monthGrid(value: string): string[][] {
  const date = parseDate(value);
  const first = new Date(date.getFullYear(), date.getMonth(), 1);
  const start = parseDate(startOfWeek(toISODate(first)));
  const weeks: string[][] = [];
  for (let w = 0; w < 6; w += 1) {
    const row: string[] = [];
    for (let d = 0; d < 7; d += 1) {
      const day = new Date(start);
      day.setDate(start.getDate() + w * 7 + d);
      row.push(toISODate(day));
    }
    weeks.push(row);
  }
  return weeks;
}

export function isSameMonth(value: string, anchor: string): boolean {
  return value.slice(0, 7) === anchor.slice(0, 7);
}

export function monthLabel(value: string): string {
  const date = parseDate(value);
  return `${date.getFullYear()} 年 ${date.getMonth() + 1} 月`;
}

export function dayLabel(value: string, weekdayNames = WEEKDAYS): string {
  const date = parseDate(value);
  const index = (date.getDay() + 6) % 7;
  return `${date.getMonth() + 1} 月 ${date.getDate()} 日 · 周${weekdayNames[index]}`;
}

export function timeOf(local: string): string {
  return local.slice(11, 16);
}

export function dateOf(local: string): string {
  return local.slice(0, 10);
}

export function monthRange(value: string): { from: string; to: string } {
  const date = parseDate(value);
  const from = new Date(date.getFullYear(), date.getMonth(), 1);
  const to = new Date(date.getFullYear(), date.getMonth() + 1, 0);
  return { from: toISODate(from), to: toISODate(to) };
}

export function weekRange(value: string): { from: string; to: string } {
  const from = startOfWeek(value);
  return { from, to: addDays(from, 6) };
}

export function agendaRange(value: string): { from: string; to: string } {
  return { from: value, to: addDays(value, 45) };
}

export function nextBirthdayDate(month: number, day: number, year: number): string {
  const last = new Date(year, month, 0).getDate();
  return `${year}-${String(month).padStart(2, '0')}-${String(Math.min(day, last)).padStart(2, '0')}`;
}

export function dateTimeToLocal(value: string): string {
  return value.length === 16 ? `${value.replace('T', ' ')}:00` : value.replace('T', ' ');
}
