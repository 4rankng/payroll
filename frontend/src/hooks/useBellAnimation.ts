import { useState, useEffect } from 'react';

export const useBellAnimation = (unreadCount: number) => {
  const [bellIcon, setBellIcon] = useState('/icons/bell.png');

  useEffect(() => {
    if (unreadCount > 0) {
      const interval = setInterval(() => {
        setBellIcon(prev =>
          prev === '/icons/bell.png' ? '/icons/bell-orange.png' : '/icons/bell.png'
        );
      }, 500);
      return () => clearInterval(interval);
    } else {
      setBellIcon('/icons/bell.png');
    }
  }, [unreadCount]);

  return bellIcon;
};