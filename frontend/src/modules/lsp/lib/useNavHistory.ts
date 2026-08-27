import { useEffect, useState } from "react";
import { getNavState, subscribe } from "./navigationHistory";

/** Reactive canGoBack/canGoForward for navigation toolbar buttons. */
export function useNavHistory(): {
  canGoBack: boolean;
  canGoForward: boolean;
} {
  const [state, setState] = useState(getNavState());
  useEffect(() => subscribe(setState), []);
  return state;
}