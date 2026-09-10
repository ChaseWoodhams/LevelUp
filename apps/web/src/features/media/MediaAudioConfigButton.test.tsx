/**
 * Tests composant — MediaAudioConfigButton (réglage des pistes audio des médias).
 * Les queries sont mockées pour piloter l'état chargé et espionner la mutation.
 */
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { screen, fireEvent } from '@testing-library/react'
import { renderWithProviders } from '@/test/render-utils'
import { MediaAudioConfigButton } from './MediaAudioConfigButton'

const mutate = vi.fn()

// Référence STABLE (module-level) : un objet littéral recréé à chaque rendu ferait
// re-tourner le seed-effect du composant en boucle (dépendance sur l'identité de data).
const AUDIO_CONFIG_DATA = { mode: 'auto' as const }

vi.mock('./queries', () => ({
  useMediaAudioConfig: () => ({ data: AUDIO_CONFIG_DATA }),
  useUpdateMediaAudioConfig: () => ({ mutate, isPending: false, isError: false, error: null }),
}))

describe('MediaAudioConfigButton', () => {
  beforeEach(() => {
    mutate.mockReset()
  })

  it('rend le bouton engrenage', () => {
    renderWithProviders(<MediaAudioConfigButton playerSlug="p1" />)
    expect(screen.getByLabelText('Audio tracks settings')).toBeInTheDocument()
  })

  it('ouvre la modale au clic', () => {
    renderWithProviders(<MediaAudioConfigButton playerSlug="p1" />)
    fireEvent.click(screen.getByLabelText('Audio tracks settings'))
    expect(screen.getByText('Media audio tracks')).toBeInTheDocument()
    expect(screen.getByText('Automatic')).toBeInTheDocument()
    expect(screen.getByText('Manual')).toBeInTheDocument()
  })

  it('affiche l’éditeur de pistes en mode manuel', () => {
    renderWithProviders(<MediaAudioConfigButton playerSlug="p1" />)
    fireEvent.click(screen.getByLabelText('Audio tracks settings'))
    fireEvent.click(screen.getByText('Manual'))
    // Seed manuel = 2 pistes (jeu + voix) + bouton d'ajout.
    expect(screen.getByText('Track 1')).toBeInTheDocument()
    expect(screen.getByText('Track 2')).toBeInTheDocument()
    expect(screen.getByText('+ Add a track')).toBeInTheDocument()
  })

  it('appelle la mutation à l’enregistrement', () => {
    renderWithProviders(<MediaAudioConfigButton playerSlug="p1" />)
    fireEvent.click(screen.getByLabelText('Audio tracks settings'))
    fireEvent.click(screen.getByText('Save'))
    expect(mutate).toHaveBeenCalledTimes(1)
    expect(mutate.mock.calls[0][0]).toMatchObject({ mode: 'auto' })
  })
})
