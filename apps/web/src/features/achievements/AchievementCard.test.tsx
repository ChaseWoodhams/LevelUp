import { describe, expect, it } from 'vitest'
import { render, screen } from '@testing-library/react'
import { AchievementCard } from './AchievementCard'
import type { AchievementEntry } from '@/lib/api/types'

const baseAchievement: AchievementEntry = {
  achievement_id: 'a',
  name_en: 'First Blood',
  description_en: 'Score your first kill',
  gamerscore: 10,
  is_secret: false,
  unlocked: false,
}

describe('AchievementCard', () => {
  it('renders the English achievement name and gamerscore', () => {
    render(<AchievementCard achievement={baseAchievement} locale="en" />)
    expect(screen.getByText('First Blood')).toBeInTheDocument()
    expect(screen.getByText('10 G')).toBeInTheDocument()
  })

  it('renders progress when current_progress and target_progress are positive', () => {
    const achievement = { ...baseAchievement, current_progress: 5, target_progress: 10 }
    render(<AchievementCard achievement={achievement} locale="en" />)
    expect(screen.getByText('5 / 10')).toBeInTheDocument()
  })

  it('renders the unlock date for an unlocked achievement', () => {
    const achievement: AchievementEntry = {
      ...baseAchievement,
      unlocked: true,
      unlocked_at: '2026-04-15T10:00:00Z',
    }
    render(<AchievementCard achievement={achievement} locale="en" />)
    expect(screen.getByText(/Unlocked on/)).toBeInTheDocument()
    expect(screen.getByText(/2026/)).toBeInTheDocument()
  })

  it('uses locked_desc_en while locked and description_en when unlocked', () => {
    const locked: AchievementEntry = { ...baseAchievement, locked_desc_en: 'Not yet earned' }
    render(<AchievementCard achievement={locked} locale="en" />)
    expect(screen.getByText('Not yet earned')).toBeInTheDocument()
    expect(screen.queryByText(baseAchievement.description_en)).not.toBeInTheDocument()

    const unlocked: AchievementEntry = { ...locked, unlocked: true }
    render(<AchievementCard achievement={unlocked} locale="en" />)
    expect(screen.getByText(baseAchievement.description_en)).toBeInTheDocument()
  })
})
