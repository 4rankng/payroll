import { Bell, KeyRound, Mail, Megaphone, MessageCircle, Receipt, Settings } from 'lucide-react';

import { TabsList, TabsTrigger } from '@/components/ui/tabs';

export const SETTINGS_TABS = {
  general: 'general',
  feeConfig: 'fee-config',
  email: 'email',
  notifications: 'notifications',
  zalo: 'zalo',
  ads: 'ads',
  apiKeys: 'api',
} as const;

export const VALID_SETTINGS_TABS = new Set<string>(Object.values(SETTINGS_TABS));

const tabs = [
  { value: SETTINGS_TABS.general, label: 'Trả lương', icon: Settings },
  { value: SETTINGS_TABS.feeConfig, label: 'Tạm ứng', icon: Receipt },
  { value: SETTINGS_TABS.email, label: 'Email', icon: Mail },
  { value: SETTINGS_TABS.notifications, label: 'Thông báo', icon: Bell },
  { value: SETTINGS_TABS.zalo, label: 'Zalo ZNS', icon: MessageCircle },
  { value: SETTINGS_TABS.ads, label: 'Quảng cáo', icon: Megaphone },
  { value: SETTINGS_TABS.apiKeys, label: 'API', icon: KeyRound },
] as const;

/**
 * Single TabsList with responsive presentation:
 * - Mobile: horizontally scrollable pill strip; active pill is filled
 *   primary with white text (iOS segment-control feel).
 * - Desktop (sm+): classic underline tabs on a full-width bottom border.
 */
export const SettingsTabList = () => (
  <div className="min-w-0">
    <TabsList className="flex h-auto w-full justify-start gap-2 overflow-x-auto overflow-y-hidden rounded-none border-b border-transparent bg-transparent p-0 pb-3 text-muted-foreground shadow-none [-webkit-overflow-scrolling:touch] [scrollbar-width:none] [&::-webkit-scrollbar]:hidden sm:flex-wrap sm:gap-1 sm:overflow-visible sm:border-border sm:pb-0 sm:w-max sm:min-w-full sm:justify-start">
      {tabs.map(({ value, label, icon: Icon }) => (
        <TabsTrigger
          key={value}
          value={value}
          className={`
            shrink-0 items-center gap-2 rounded-full border border-transparent bg-muted/60 px-4 py-2.5 text-sm font-medium shadow-none transition-colors
            data-[state=active]:border-primary/20 data-[state=active]:bg-primary data-[state=active]:text-primary-foreground data-[state=active]:shadow-sm
            hover:text-foreground
            sm:shrink sm:gap-2 sm:rounded-none sm:border-transparent sm:bg-transparent sm:px-4 sm:py-2.5
            sm:min-h-11 sm:data-[state=active]:border-primary sm:data-[state=active]:bg-transparent sm:data-[state=active]:text-foreground sm:data-[state=active]:shadow-none
          `}
        >
          <Icon aria-hidden="true" className="h-4 w-4 shrink-0" />
          {label}
        </TabsTrigger>
      ))}
    </TabsList>
  </div>
);
