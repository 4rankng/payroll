import { KeyRound, Megaphone, MessageCircle, Receipt, Settings, Users } from 'lucide-react';

import { TabsList, TabsTrigger } from '@/components/ui/tabs';

export const SETTINGS_TABS = {
  general: 'general',
  feeConfig: 'fee-config',
  employees: 'employees',
  zalo: 'zalo',
  ads: 'ads',
  apiKeys: 'api',
} as const;

export const VALID_SETTINGS_TABS = new Set<string>(Object.values(SETTINGS_TABS));

const tabs = [
  { value: SETTINGS_TABS.general, label: 'Trả lương', icon: Settings },
  { value: SETTINGS_TABS.feeConfig, label: 'Tạm ứng', icon: Receipt },
  { value: SETTINGS_TABS.employees, label: 'Nhân viên', icon: Users },
  { value: SETTINGS_TABS.zalo, label: 'Zalo ZNS', icon: MessageCircle },
  { value: SETTINGS_TABS.ads, label: 'Quảng cáo', icon: Megaphone },
  { value: SETTINGS_TABS.apiKeys, label: 'API', icon: KeyRound },
] as const;

/**
 * One compact console tab strip: a left-aligned track, 36px desktop / 44px
 * mobile targets, an active card with a hairline ring. Below `sm` it scrolls
 * horizontally instead of wrapping into two rows.
 */
export const SettingsTabList = () => (
  <div className="min-w-0 overflow-x-auto [-webkit-overflow-scrolling:touch] [scrollbar-width:none] [&::-webkit-scrollbar]:hidden">
    <TabsList className="inline-flex h-auto w-max min-h-0 items-center justify-start gap-1 rounded-lg border border-border bg-muted/40 p-1 text-muted-foreground">
      {tabs.map(({ value, label, icon: Icon }) => (
        <TabsTrigger
          key={value}
          value={value}
          className="inline-flex h-11 min-h-11 shrink-0 items-center justify-center gap-1.5 rounded-md px-2.5 py-1.5 text-[12px] font-medium text-muted-foreground shadow-none transition-colors duration-100 hover:text-foreground data-[state=active]:bg-card data-[state=active]:text-foreground data-[state=active]:shadow-xs data-[state=active]:ring-1 data-[state=active]:ring-border sm:h-9 sm:min-h-9 sm:px-3 sm:text-[13px]"
        >
          <Icon aria-hidden="true" className="hidden h-4 w-4 shrink-0 sm:block" />
          {label}
        </TabsTrigger>
      ))}
    </TabsList>
  </div>
);
