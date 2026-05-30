import {
  createContext,
  useContext,
  useState,
  useCallback,
  useMemo,
  type ReactNode,
} from "react";

interface BottomNavOverride {
  content: ReactNode;
}

interface BottomNavContextValue {
  override: BottomNavOverride | null;
  setOverride: (override: BottomNavOverride | null) => void;
}

const BottomNavContext = createContext<BottomNavContextValue>({
  override: null,
  setOverride: () => {},
});

export function BottomNavProvider({ children }: { children: ReactNode }) {
  const [override, setOverrideState] = useState<BottomNavOverride | null>(null);

  const setOverride = useCallback((o: BottomNavOverride | null) => {
    setOverrideState(o);
  }, []);

  const value = useMemo(
    () => ({ override, setOverride }),
    [override, setOverride],
  );

  return (
    <BottomNavContext.Provider value={value}>
      {children}
    </BottomNavContext.Provider>
  );
}

export function useBottomNav() {
  return useContext(BottomNavContext);
}
