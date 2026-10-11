import { useState } from 'react';
import { MessageChatCircle } from '@untitledui/icons';

import { AssistantPanel } from '@/components/chat/AssistantPanel';
import { useSettingByKey } from '@/hooks/api/useSettings';
import { MIMO_SETTING_KEYS } from '@/types/api/settings.types';
import { useIsMobile } from '@/hooks/useBreakpoint';
import { useUnreadNotifications } from '@/hooks/api/useNotifications';
import { authManager } from '@/lib/auth';

/** Floating trigger for the MiMo assistant. Rendered for admin + partner only
 * (adv_partner shares the admin layout but is excluded), and hidden when the
 * admin switched the role off in settings. Desktop sits above the
 * notification bell slot; mobile owns bottom-right (bell hides on mobile). */
export const AssistantFAB = () => {
  const isMobile = useIsMobile();
  const [open, setOpen] = useState(false);

  const role = authManager.getUserRole();
  const settingKey =
    role === 'admin' ? MIMO_SETTING_KEYS.chatEnabledAdmin : MIMO_SETTING_KEYS.chatEnabledPartner;
  const { data: setting, isLoading } = useSettingByKey(settingKey);
  // Anchor lower-right; stack above the bell only while it is visible.
  const { data: unreadData } = useUnreadNotifications();
  const bellVisible = (unreadData?.count ?? 0) > 0;

  if (role !== 'admin' && role !== 'partner') return null;
  if (isLoading || setting?.value === 'false') return null;

  return (
    <>
      <button
        type="button"
        onClick={() => setOpen(true)}
        className={`fixed right-4 z-50 flex h-12 w-12 items-center justify-center rounded-full bg-indigo-600 text-white shadow-sm transition-all duration-300 hover:bg-indigo-700 active:scale-95 animate-in slide-in-from-bottom-4 fade-in ${bellVisible ? 'bottom-24' : 'bottom-6'}`}
        aria-label="Mở trợ lý AI"
      >
        <MessageChatCircle className="size-5" aria-hidden="true" />
      </button>
      <AssistantPanel open={open} onClose={() => setOpen(false)} />
    </>
  );
};
