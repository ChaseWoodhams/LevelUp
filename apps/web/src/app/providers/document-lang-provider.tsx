import { type ReactNode, useLayoutEffect } from 'react'
import { useAppShellStore } from '@/stores/appShellStore'
import { intlLocale } from '@/lib/formatters/intlLocale'

/**
 * DocumentLangProvider keeps the document language aligned with the English-only
 * application runtime.
 */
export function DocumentLangProvider({ children }: { children: ReactNode }) {
  const locale = useAppShellStore((s) => s.locale)

  useLayoutEffect(() => {
    document.documentElement.lang = intlLocale()
  }, [locale])

  return <>{children}</>
}
