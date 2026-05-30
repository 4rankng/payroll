import { useIsMobile } from '@/hooks/use-mobile';

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
    <div className="mobile-page">
      <Mobile />
    </div>
  ) : <Desktop />;
};
