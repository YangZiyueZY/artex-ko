import * as React from "react";

const MOBILE_BREAKPOINT = 768;
// 侧边栏展开时若正文宽度变窄，表格·卡片会重叠（768~1279px 区间）。
// 低于该宽度时自动把侧边栏折叠为图标模式，以保证正文宽度。
export const TABLET_BREAKPOINT = 1280;

function useBreakpoint(breakpoint: number) {
  const [below, setBelow] = React.useState<boolean | undefined>(undefined);

  React.useEffect(() => {
    const mql = window.matchMedia(`(max-width: ${breakpoint - 1}px)`);
    const onChange = () => setBelow(window.innerWidth < breakpoint);
    mql.addEventListener("change", onChange);
    setBelow(window.innerWidth < breakpoint);
    return () => mql.removeEventListener("change", onChange);
  }, [breakpoint]);

  return below;
}

export function useIsMobile() {
  return !!useBreakpoint(MOBILE_BREAKPOINT);
}

// 是否为平板宽度（768~1279px）。初始渲染（测量前）原样返回 undefined，
// 以便调用方区分“尚未可知”与“false”。
export function useIsTablet() {
  return useBreakpoint(TABLET_BREAKPOINT);
}
