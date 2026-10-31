import { createContext, useContext } from 'react';

export interface ConfirmOptions {
  title?: string;
  message: string;
  confirmLabel?: string;
  danger?: boolean;
}

interface ConfirmContextType {
  confirm: (options: ConfirmOptions) => Promise<boolean>;
}

export const ConfirmContext = createContext<ConfirmContextType>({ confirm: () => Promise.resolve(false) });

export function useConfirm() {
  return useContext(ConfirmContext);
}
