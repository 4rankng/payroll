import { useIsMobile } from '@/hooks/useBreakpoint';

interface ResponsivePageProps {
  desktopComponent: React.ComponentType;
  mobileComponent: React.ComponentType;
}

/**
 * Component that conditionally renders desktop or mobile version of a page
 * based on screen size using the useIsMobile hook
 */
export const ResponsivePage = ({ desktopComponent: Desktop, mobileComponent: Mobile }: ResponsivePageProps) => {
  const isMobile = useIsMobile();

  return isMobile ? (
    <div className="mobile-page min-h-[100dvh] w-[100cqw] max-w-none overflow-x-clip bg-[hsl(var(--surface-page))] text-slate-950">
      <Mobile />
    </div>
  ) : <Desktop />;
};
