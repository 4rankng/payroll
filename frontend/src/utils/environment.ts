export const isDevelopmentEnvironment = import.meta.env.DEV;
export const environmentMode = import.meta.env.MODE;

export const shouldShowDevelopmentBanner = false;

export const developmentBannerText: { headline: string; detail?: string } = {
  headline: "Đang chạy trên môi trường phát triển",
};
