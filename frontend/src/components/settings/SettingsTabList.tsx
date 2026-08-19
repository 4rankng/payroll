import { Bell, Mail, MessageCircle, Receipt, Settings } from 'lucide-react';

import { TabsList, TabsTrigger } from '@/components/ui/tabs';

export const SETTINGS_TABS = {
  general: 'general',
  feeConfig: 'fee-config',
  email: 'email',
  notifications: 'notifications',
  zalo: 'zalo',
} as const;

export const VALID_SETTINGS_TABS = new Set<string>(Object.values(SETTINGS_TABS));

const tabs = [
  { value: SETTINGS_TABS.general, label: 'Trả lương', icon: Settings, mobileSpan: 'col-span-2' },
  { value: SETTINGS_TABS.feeConfig, label: 'Tạm ứng', icon: Receipt, mobileSpan: 'col-span-2' },
  { value: SETTINGS_TABS.email, label: 'Email', icon: Mail, mobileSpan: 'col-span-2' },
  {
    value: SETTINGS_TABS.notifications,
    label: 'Thông báo',
    icon: Bell,
    mobileSpan: 'col-span-3',
  },
  { value: SETTINGS_TABS.zalo, label: 'Zalo ZNS', icon: MessageCircle, mobileSpan: 'col-span-3' },
] as const;

export const SettingsTabList = () => (
  <div className="min-w-0">
    <TabsList className="grid h-auto w-full grid-cols-6 justify-start rounded-none border-b bg-transparent p-0 text-muted-foreground shadow-none sm:flex sm:w-max sm:min-w-full sm:gap-1">
      {tabs.map(({ value, label, icon: Icon, mobileSpan }) => (
        <TabsTrigger
          key={value}
          value={value}
          className={`${mobileSpan} min-h-11 gap-1.5 rounded-none border-b-2 border-transparent px-2 py-2.5 text-sm shadow-none transition-colors hover:text-foreground data-[state=active]:border-primary data-[state=active]:bg-transparent data-[state=active]:text-foreground data-[state=active]:shadow-none sm:col-auto sm:gap-2 sm:px-4`}
        >
          <Icon aria-hidden="true" className="h-4 w-4 shrink-0" />
          {label}
        </TabsTrigger>
      ))}
    </TabsList>
  </div>
);
