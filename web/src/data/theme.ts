import type { CardRadius, CardTone, ThemeConfig, ThemeTone } from '@/types';

function svgUrl(svg: string): string {
  return `data:image/svg+xml,${encodeURIComponent(svg.replace(/\s{2,}/g, ' ').trim())}`;
}

export interface TonePreset {
  key: ThemeTone;
  label: string;
  group: 'basic' | 'warm' | 'cartoon';
  swatch: string;
  description: string;
}

export const tonePresets: TonePreset[] = [
  { key: 'default', label: '默认', group: 'basic', swatch: '#f5f5f5', description: '编辑部白' },
  { key: 'warm-sun', label: '暖阳', group: 'warm', swatch: '#f7efe2', description: '浅杏暖光' },
  { key: 'warm-sand', label: '沙色', group: 'warm', swatch: '#f1e8db', description: '暖灰褐' },
  { key: 'warm-peach', label: '蜜桃', group: 'warm', swatch: '#f9ece2', description: '粉杏' },
  { key: 'warm-cream', label: '奶咖', group: 'warm', swatch: '#f5ece0', description: '奶油咖啡' },
  { key: 'warm-rose', label: '暖粉', group: 'warm', swatch: '#f9e9ec', description: '柔和粉' },
  { key: 'cartoon-macaron', label: '马卡龙', group: 'cartoon', swatch: '#f6eef2', description: '粉薄荷混搭' },
  { key: 'cartoon-sky', label: '天空蓝', group: 'cartoon', swatch: '#e9f1f8', description: '淡蓝' },
  { key: 'cartoon-mint', label: '薄荷绿', group: 'cartoon', swatch: '#eaf4ef', description: '淡绿' },
  { key: 'cartoon-berry', label: '浆果粉', group: 'cartoon', swatch: '#f7e6ec', description: '莓粉' },
  { key: 'cartoon-lemon', label: '柠檬黄', group: 'cartoon', swatch: '#f8f2dd', description: '淡黄' },
];

export const radiusOptions: { key: CardRadius; label: string; px: number }[] = [
  { key: 'sm', label: '小', px: 8 },
  { key: 'md', label: '中', px: 12 },
  { key: 'lg', label: '大', px: 16 },
  { key: 'xl', label: '超大', px: 24 },
];

export const radiusPx: Record<CardRadius, number> = { sm: 8, md: 12, lg: 16, xl: 24 };

export interface CardTonePreset {
  key: CardTone;
  label: string;
  swatch: string;
}

export const cardTonePresets: CardTonePreset[] = [
  { key: 'default', label: '默认', swatch: '#ffffff' },
  { key: 'soft', label: '浅灰', swatch: '#fafafa' },
  { key: 'sand', label: '沙色', swatch: '#f7f4ef' },
  { key: 'mint', label: '薄荷', swatch: '#eef7f3' },
  { key: 'peach', label: '蜜桃', swatch: '#fbf2ec' },
  { key: 'lavender', label: '薰衣草', swatch: '#f4f0f9' },
  { key: 'sky', label: '天空', swatch: '#edf3fa' },
  { key: 'rose', label: '玫瑰', swatch: '#faf0f2' },
];

export interface BackgroundPreset {
  key: string;
  label: string;
  swatch: string;
  image: string;
}

const bgWarmGlow = svgUrl(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 800 600">
  <defs><radialGradient id="g" cx="22%" cy="18%" r="70%">
    <stop offset="0%" stop-color="#ffe9c9"/><stop offset="55%" stop-color="#f9e3d3"/><stop offset="100%" stop-color="#f3ece4"/>
  </radialGradient></defs>
  <rect width="800" height="600" fill="url(#g)"/>
  <circle cx="640" cy="120" r="96" fill="#ffd9a8" opacity="0.55"/>
  <circle cx="700" cy="420" r="150" fill="#f7c9b0" opacity="0.4"/>
  <circle cx="140" cy="500" r="120" fill="#f6dfc2" opacity="0.5"/>
</svg>`);

const bgSandHills = svgUrl(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 800 600">
  <rect width="800" height="600" fill="#f3ead9"/>
  <path d="M0 420 C 180 340 320 470 460 400 C 600 330 700 400 800 360 L800 600 L0 600 Z" fill="#e9dcc5" opacity="0.85"/>
  <path d="M0 500 C 160 440 300 540 470 480 C 620 428 720 500 800 470 L800 600 L0 600 Z" fill="#e0d0b4" opacity="0.8"/>
  <circle cx="650" cy="140" r="72" fill="#f6e7c8" opacity="0.85"/>
</svg>`);

const bgPeachBlossom = svgUrl(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 800 600">
  <rect width="800" height="600" fill="#fbece4"/>
  <g fill="#f6c9b6" opacity="0.6">
    <circle cx="120" cy="120" r="46"/><circle cx="190" cy="90" r="26"/><circle cx="90" cy="200" r="30"/>
    <circle cx="680" cy="180" r="54"/><circle cx="610" cy="110" r="24"/>
    <circle cx="700" cy="470" r="40"/><circle cx="620" cy="520" r="22"/>
  </g>
  <g fill="#f3b7a2" opacity="0.5">
    <circle cx="240" cy="420" r="34"/><circle cx="300" cy="470" r="20"/>
    <circle cx="520" cy="300" r="26"/>
  </g>
</svg>`);

const bgSkyCloud = svgUrl(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 800 600">
  <defs><linearGradient id="s" x1="0" y1="0" x2="0" y2="1">
    <stop offset="0%" stop-color="#e3eef9"/><stop offset="100%" stop-color="#f2f7fc"/>
  </linearGradient></defs>
  <rect width="800" height="600" fill="url(#s)"/>
  <g fill="#ffffff" opacity="0.8">
    <ellipse cx="180" cy="150" rx="120" ry="46"/><ellipse cx="260" cy="140" rx="80" ry="34"/>
    <ellipse cx="600" cy="300" rx="140" ry="50"/><ellipse cx="520" cy="290" rx="70" ry="30"/>
    <ellipse cx="300" cy="470" rx="110" ry="38"/>
  </g>
</svg>`);

const bgMintLeaf = svgUrl(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 800 600">
  <rect width="800" height="600" fill="#ecf6f0"/>
  <g fill="none" stroke="#b9dfc9" stroke-width="6" opacity="0.7">
    <path d="M120 560 C 200 420 190 300 300 200"/>
    <path d="M300 200 C 350 250 340 320 280 340"/>
    <path d="M620 540 C 600 420 660 340 720 300"/>
    <path d="M720 300 C 720 380 660 420 610 400"/>
  </g>
  <g fill="#c9e8d6" opacity="0.6">
    <ellipse cx="210" cy="330" rx="54" ry="26" transform="rotate(-32 210 330)"/>
    <ellipse cx="660" cy="380" rx="60" ry="26" transform="rotate(24 660 380)"/>
  </g>
</svg>`);

const bgBerryDots = svgUrl(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 800 600">
  <rect width="800" height="600" fill="#f9eaf0"/>
  <g fill="#f0c3d2" opacity="0.7">
    <circle cx="110" cy="120" r="34"/><circle cx="230" cy="240" r="20"/><circle cx="360" cy="110" r="26"/>
    <circle cx="520" cy="220" r="38"/><circle cx="680" cy="120" r="22"/>
    <circle cx="160" cy="450" r="28"/><circle cx="330" cy="500" r="18"/>
    <circle cx="600" cy="470" r="34"/><circle cx="730" cy="380" r="18"/>
  </g>
</svg>`);

export const backgroundPresets: BackgroundPreset[] = [
  { key: 'bg-warm-glow', label: '暖光', swatch: '#ffe9c9', image: bgWarmGlow },
  { key: 'bg-sand-hills', label: '沙丘', swatch: '#e9dcc5', image: bgSandHills },
  { key: 'bg-peach-blossom', label: '桃枝', swatch: '#f6c9b6', image: bgPeachBlossom },
  { key: 'bg-sky-cloud', label: '云朵', swatch: '#e3eef9', image: bgSkyCloud },
  { key: 'bg-mint-leaf', label: '薄荷叶', swatch: '#c9e8d6', image: bgMintLeaf },
  { key: 'bg-berry-dots', label: '莓果点', swatch: '#f0c3d2', image: bgBerryDots },
];

export const backgroundByKey: Record<string, string> = Object.fromEntries(
  backgroundPresets.map((item) => [item.key, item.image]),
);

export interface CornerPreset {
  key: string;
  label: string;
  group: 'pattern' | 'character';
  image: string;
}

const cornerSvg = (inner: string): string => svgUrl(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 120 120">${inner}</svg>`);

export const cornerPresets: CornerPreset[] = [
  {
    key: 'corner-vine',
    label: '藤蔓',
    group: 'pattern',
    image: cornerSvg(`<g fill="none" stroke="#7a6a58" stroke-width="3" stroke-linecap="round" opacity="0.9">
      <path d="M118 108 C 92 104 78 88 82 66 C 86 46 70 34 52 40"/>
      <path d="M82 66 C 94 64 102 72 104 84"/>
      <path d="M70 50 C 66 36 74 26 88 24"/>
      <path d="M52 40 C 38 44 28 34 30 20"/>
    </g>`),
  },
  {
    key: 'corner-leaf',
    label: '叶片',
    group: 'pattern',
    image: cornerSvg(`<g fill="#7a6a58" opacity="0.85">
      <path d="M116 116 C 80 112 58 88 62 58 C 92 60 114 82 116 116 Z"/>
      <path d="M104 104 C 92 90 84 78 80 66" fill="none" stroke="#fff" stroke-width="2" opacity="0.5"/>
      <path d="M52 64 C 38 60 30 48 32 34 C 46 36 56 48 52 64 Z" opacity="0.6"/>
    </g>`),
  },
  {
    key: 'corner-wave',
    label: '波纹',
    group: 'pattern',
    image: cornerSvg(`<g fill="none" stroke="#7a6a58" stroke-width="3" stroke-linecap="round" opacity="0.8">
      <path d="M6 104 C 26 84 46 84 66 104 C 86 124 106 124 126 104"/>
      <path d="M6 84 C 26 64 46 64 66 84 C 86 104 106 104 126 84"/>
      <path d="M6 64 C 26 44 46 44 66 64 C 86 84 106 84 126 64"/>
    </g>`),
  },
  {
    key: 'corner-dot',
    label: '星点',
    group: 'pattern',
    image: cornerSvg(`<g fill="#7a6a58" opacity="0.8">
      <circle cx="104" cy="104" r="6"/><circle cx="80" cy="92" r="3"/><circle cx="92" cy="72" r="3"/>
      <circle cx="66" cy="104" r="3"/><circle cx="106" cy="80" r="3"/>
      <path d="M110 56 l3 7 7 3 -7 3 -3 7 -3 -7 -7 -3 7 -3 z" opacity="0.7"/>
    </g>`),
  },
  {
    key: 'corner-bear',
    label: '小熊',
    group: 'character',
    image: cornerSvg(`<g fill="#8a6a4f" opacity="0.85">
      <circle cx="46" cy="34" r="10"/><circle cx="82" cy="34" r="10"/>
      <circle cx="64" cy="62" r="30"/>
      <ellipse cx="64" cy="74" rx="16" ry="12" fill="#f3e6d7"/>
      <circle cx="54" cy="56" r="3" fill="#3a2d24"/><circle cx="74" cy="56" r="3" fill="#3a2d24"/>
      <ellipse cx="64" cy="66" rx="4" ry="3" fill="#3a2d24"/>
      <path d="M64 70 c -3 4 -8 4 -10 1 M64 70 c 3 4 8 4 10 1" fill="none" stroke="#3a2d24" stroke-width="2" stroke-linecap="round"/>
    </g>`),
  },
  {
    key: 'corner-rabbit',
    label: '兔子',
    group: 'character',
    image: cornerSvg(`<g fill="#8a6a4f" opacity="0.85">
      <ellipse cx="50" cy="30" rx="9" ry="20"/><ellipse cx="78" cy="30" rx="9" ry="20"/>
      <circle cx="64" cy="66" r="26"/>
      <circle cx="55" cy="62" r="3" fill="#3a2d24"/><circle cx="73" cy="62" r="3" fill="#3a2d24"/>
      <path d="M64 70 l-4 4 M64 70 l4 4" stroke="#3a2d24" stroke-width="2" stroke-linecap="round"/>
      <ellipse cx="64" cy="74" rx="3" ry="2" fill="#c98f8f"/>
    </g>`),
  },
  {
    key: 'corner-cat',
    label: '小猫',
    group: 'character',
    image: cornerSvg(`<g fill="#8a6a4f" opacity="0.85">
      <path d="M38 40 L44 18 L58 32 Z"/><path d="M90 40 L84 18 L70 32 Z"/>
      <circle cx="64" cy="62" r="27"/>
      <circle cx="54" cy="58" r="3" fill="#3a2d24"/><circle cx="74" cy="58" r="3" fill="#3a2d24"/>
      <path d="M64 68 l-3 3 M64 68 l3 3" stroke="#3a2d24" stroke-width="2" stroke-linecap="round"/>
      <g stroke="#3a2d24" stroke-width="2" stroke-linecap="round">
        <path d="M34 66 h12 M34 74 h12 M94 66 h-12 M94 74 h-12"/>
      </g>
    </g>`),
  },
  {
    key: 'corner-cloud',
    label: '云朵',
    group: 'character',
    image: cornerSvg(`<g fill="#8a6a4f" opacity="0.75">
      <ellipse cx="58" cy="62" rx="26" ry="18"/><circle cx="46" cy="56" r="14"/><circle cx="66" cy="50" r="16"/><circle cx="78" cy="60" r="12"/>
      <circle cx="52" cy="64" r="2" fill="#3a2d24"/><circle cx="66" cy="64" r="2" fill="#3a2d24"/>
      <path d="M58 70 c 2 3 5 3 7 0" fill="none" stroke="#3a2d24" stroke-width="2" stroke-linecap="round"/>
    </g>`),
  },
  {
    key: 'corner-sun',
    label: '太阳',
    group: 'character',
    image: cornerSvg(`<g fill="#8a6a4f" opacity="0.8">
      <circle cx="64" cy="64" r="20"/>
      <g stroke="#8a6a4f" stroke-width="4" stroke-linecap="round">
        <path d="M64 30 v-12 M64 110 v-12 M30 64 h-12 M110 64 h-12"/>
        <path d="M40 40 l-8 -8 M96 96 l-8 -8 M40 88 l-8 8 M96 32 l-8 8"/>
      </g>
      <circle cx="57" cy="62" r="2.5" fill="#3a2d24"/><circle cx="71" cy="62" r="2.5" fill="#3a2d24"/>
      <path d="M58 70 c 2 3 5 3 7 0" fill="none" stroke="#3a2d24" stroke-width="2" stroke-linecap="round"/>
    </g>`),
  },
];

export const cornerByKey: Record<string, CornerPreset> = Object.fromEntries(
  cornerPresets.map((item) => [item.key, item]),
);

export const SYSTEM_THEME: ThemeConfig = {
  scheme: 'light',
  cardTone: 'default',
  tone: 'default',
  cardRadius: 'lg',
  pageBackground: { kind: 'color', blur: 8 },
  cardCorner: { image: null, opacity: 0.12 },
};

export const DEFAULT_BACKGROUND_COLOR = '#f5f5f5';
