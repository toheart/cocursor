import { useEffect, useRef } from "react";

/**
 * 跟踪组件挂载状态的 Hook
 * 用于防止在组件卸载后更新状态
 */
export const useMounted = (): React.MutableRefObject<boolean> => {
  const isMountedRef = useRef(true);

  useEffect(() => {
    isMountedRef.current = true;
    return () => {
      isMountedRef.current = false;
    };
  }, []);

  return isMountedRef;
};
