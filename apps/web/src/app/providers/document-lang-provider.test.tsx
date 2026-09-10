/** The document language is always English. */
import { describe, it, expect, afterEach } from 'vitest'
import { render } from '@testing-library/react'
import { useAppShellStore } from '@/stores/appShellStore'
import { DocumentLangProvider } from './document-lang-provider'

afterEach(() => {
  useAppShellStore.setState({ locale: 'en' })
  document.documentElement.removeAttribute('lang')
})

describe('DocumentLangProvider', () => {
  it('sets lang="en-US"', () => {
    useAppShellStore.setState({ locale: 'en' })
    render(
      <DocumentLangProvider>
        <span />
      </DocumentLangProvider>,
    )
    expect(document.documentElement.lang).toBe('en-US')
  })

})
