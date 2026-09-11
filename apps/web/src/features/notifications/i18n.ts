/**
 * i18n FR/EN du système de notifications.
 *
 * Pattern : dictionnaire typé par locale, accessible via getNotificationsText(locale).
 * Aligné avec features/{settings,help,...}/i18n.ts.
 *
 * Les clés notif.<category>.title et notif.<category>.body correspondent aux
 * title_key/body_key reçus du backend. Résolues par format.ts en injectant
 * les params (interpolation simple {name}, {count} sans plurals ICU pour le MVP).
 */
import type { NotificationCategory } from './types'
import type { Locale } from '@/lib/i18n/locale'

export interface NotificationsText {
  // Cloche & badge
  bellAriaLabelEmpty: string
  bellAriaLabelWithCount: string // "{count} notifications non lues"

  // Dropdown
  dropdownTitle: string
  dropdownUnread: string
  dropdownOlder: string
  dropdownEmpty: string
  dropdownMarkAllRead: string
  dropdownViewAll: string
  dropdownErrorLoading: string

  // Actions item
  actionMarkAsRead: string
  actionMarkAsUnread: string
  actionDismiss: string
  actionView: string

  // Page dédiée
  pageTitle: string
  pageFilterAll: string
  pageFilterUnread: string
  pageGroupToday: string
  pageGroupYesterday: string
  pageGroupThisWeek: string
  pageGroupOlder: string
  pageBulkMarkRead: string
  pageBulkDismiss: string
  pageLoadMore: string
  pageEmpty: string

  // Settings tab
  settingsTitle: string
  settingsDescription: string
  settingsMaster: string
  settingsMasterDescription: string
  settingsToasts: string
  settingsToastsDescription: string
  settingsCategoriesTitle: string
  settingsCategoriesDescription: string
  settingsDeliveryBoth: string
  settingsDeliveryInApp: string
  settingsDeliveryToast: string
  settingsDeliveryOff: string
  settingsSubscriptionsTitle: string
  settingsSubscriptionsDescription: string
  settingsRetentionTitle: string
  settingsRetentionLabel: string
  settingsTestButton: string
  settingsTestSent: string

  // Catégories — label + description (pour Settings)
  categoryLabel: Record<NotificationCategory, string>
  categoryDescription: Record<NotificationCategory, string>

  // Mapping clé métrique (envoyée par le backend) → libellé localisé.
  // Sert à éviter tout hardcode "KD"/"Winrate" dans les templates.
  metricLabel: Record<string, string>

  // Mapping fenêtre temporelle records V2 → libellé localisé (30d/90d/all_time).
  periodLabel: Record<string, string>

  // Templates de notif (rendus à partir de title_key/body_key + params)
  // Convention : "notif.<category>.title" / ".body"
  // Gardé sous forme de Record<string,string> pour permettre extension future
  // sans modifier le typage.
  templates: Record<string, string>

  // Temps relatif
  relJustNow: string
  relMinutesAgo: (n: number) => string
  relHoursAgo: (n: number) => string
  relDaysAgo: (n: number) => string
  relOnDate: (iso: string) => string
}

const EN: NotificationsText = {
  bellAriaLabelEmpty: 'No unread notifications',
  bellAriaLabelWithCount: '{count} unread notifications',

  dropdownTitle: 'Notifications',
  dropdownUnread: 'Unread',
  dropdownOlder: 'Recent',
  dropdownEmpty: "You're all caught up",
  dropdownMarkAllRead: 'Mark all as read',
  dropdownViewAll: 'View all',
  dropdownErrorLoading: 'Failed to load notifications',

  actionMarkAsRead: 'Mark as read',
  actionMarkAsUnread: 'Mark as unread',
  actionDismiss: 'Dismiss',
  actionView: 'View',

  pageTitle: 'Notifications',
  pageFilterAll: 'All',
  pageFilterUnread: 'Unread only',
  pageGroupToday: 'Today',
  pageGroupYesterday: 'Yesterday',
  pageGroupThisWeek: 'This week',
  pageGroupOlder: 'Older',
  pageBulkMarkRead: 'Mark as read ({count})',
  pageBulkDismiss: 'Dismiss ({count})',
  pageLoadMore: 'Load more',
  pageEmpty: 'No notifications yet',

  settingsTitle: 'Notifications',
  settingsDescription:
    'Manage in-app notifications, pop-ups and per-category subscriptions.',
  settingsMaster: 'Enable notifications',
  settingsMasterDescription:
    'Fully disables bell and pop-ups. Events still get persisted server-side.',
  settingsToasts: 'Enable pop-ups',
  settingsToastsDescription:
    'Show new notifications as on-screen pop-ups (in addition to the bell panel).',
  settingsCategoriesTitle: 'Categories',
  settingsCategoriesDescription:
    'Pick how you want to be notified for each event type.',
  settingsDeliveryBoth: 'Bell panel + pop-up',
  settingsDeliveryInApp: 'Bell panel only',
  settingsDeliveryToast: 'Pop-up only',
  settingsDeliveryOff: 'Off',
  settingsSubscriptionsTitle: 'Subscriptions',
  settingsSubscriptionsDescription:
    'Toggle automatic tracking for the current player.',
  settingsRetentionTitle: 'Retention',
  settingsRetentionLabel: 'Keep last {n} notifications',
  settingsTestButton: 'Send test notification',
  settingsTestSent: 'Test notification sent',

  categoryLabel: {
    app_release: 'App release',
    match_synced: 'Match synced',
    media_added: 'Media added',
    media_liked: 'Media liked',
    objective_assigned: 'New objective',
    objective_completed: 'Objective completed',
    challenge_added: 'New challenge',
    challenge_completed: 'Challenge completed',
    season_pass_level: 'Season pass level (deprecated)',
    sync_error: 'Sync error',
    personal_record: 'Personal record',
    threshold_crossed: 'Threshold crossed',
    friend_added: 'Friend added',
    friend_sync_completed: 'Friend sessions updated',
    data_health_warning: 'Database audit',
    career_rank: 'Career rank',
    skill_tier: 'CSR / LUSR tier',
    battlepass_completed: 'Battle pass completed',
    citation_tier: 'Commendation tier',
    citation_mastery: 'Commendation mastered',
    record_near_miss: 'Record near miss',
    milestone_unlocked: 'Milestone unlocked',
    milestone_near_miss: 'Milestone near miss',
    lusr_tier_approach: 'LUSR tier approach',
    streak_milestone: 'Streak milestone',
    comeback_welcome: 'Welcome back',
    trend_consolidate: 'Focus to consolidate',
    title_ready: 'Title ready',
    rival_encounter: 'Rival encountered',
    medal_first_earned: 'First-time medal',
  },
  categoryDescription: {
    app_release: 'A new LevelUp version is available.',
    match_synced: 'New matches were synchronized.',
    media_added: 'A video or screenshot was added.',
    media_liked: 'Someone liked one of your media.',
    objective_assigned: 'An objective was auto-assigned to you.',
    objective_completed: 'You completed an objective.',
    challenge_added: 'A new challenge is available.',
    challenge_completed: 'You completed a daily or weekly challenge.',
    season_pass_level: 'Legacy category superseded by "Career rank" and "Battle pass completed".',
    sync_error: 'The sync failed — manual retry recommended.',
    personal_record: 'You broke a personal record.',
    threshold_crossed: 'A K/D or winrate threshold was crossed.',
    friend_added: 'A gamertag was added to your friends list.',
    friend_sync_completed: 'Matches were reclassified as squad after a friend addition.',
    data_health_warning: 'Anomalies detected by the periodic DB audit (raw UUIDs, lying bits, stale URLs).',
    career_rank: 'You earned a new Halo career rank (lifetime).',
    skill_tier: 'Your competitive tier changed on a playlist (CSR or LUSR).',
    battlepass_completed: 'You completed a battle pass (max rank reached).',
    citation_tier: 'You crossed a new tier on a commendation.',
    citation_mastery: 'You mastered a commendation to 100%.',
    record_near_miss: 'Your current score is approaching one of your personal bests.',
    milestone_unlocked: 'You just unlocked a milestone (cumulative threshold).',
    milestone_near_miss: 'You are close to unlocking a milestone.',
    lusr_tier_approach: 'Your LUSR rating is approaching the next sub-tier.',
    streak_milestone: 'Your streak hit a milestone (PP multiplier).',
    comeback_welcome: 'You are back after a pause — welcome!',
    trend_consolidate: 'One of your performance areas has been trending down over time — a chance to shore it up.',
    title_ready: 'A newly activated title finished its first sync.',
    rival_encounter: 'A sync brought a new duel against one of your top rivals.',
    medal_first_earned: 'You earned a medal you had never obtained before.',
  },

  metricLabel: {
    kd_ratio: 'K/D ratio',
    winrate: 'winrate',
    kda: 'KDA',
    // Progression V2 (sent by the coach generator under `metric` key).
    performance_score: 'performance score',
    kpm: 'kills per minute',
    accuracy: 'accuracy',
    pspm: 'personal score per minute',
  },

  periodLabel: {
    '30d': '30 days',
    '90d': '90 days',
    all_time: 'all-time',
  },

  templates: {
    'notif.app_release.title': 'New version: {version}',
    'notif.app_release.body': 'See what changed in the changelog.',
    'notif.match_synced.title': '{count} match(es) synced',
    'notif.match_synced.body': 'Your stats are up to date.',
    'notif.media_added.title': 'Media added by {actor_name}',
    'notif.media_added.body': '{count} file(s) linked to a match.',
    'notif.media_liked.title': '{actor_name} liked your media',
    'notif.media_liked.body': 'Check it out in the gallery.',
    'notif.objective_assigned.title': 'New objective',
    'notif.objective_assigned.body': '{count} new objective(s) assigned.',
    'notif.objective_completed.title': 'Objective completed',
    'notif.objective_completed.body': '{count} objective(s) completed — nice!',
    'notif.challenge_added.title': 'New challenge(s) available',
    'notif.challenge_added.body': '{count} new challenge(s).',
    'notif.challenge_completed.title': 'Challenge completed',
    'notif.challenge_completed.body': '{count} citation(s) earned.',
    'notif.season_pass_level.title': 'Level {level} reached',
    'notif.season_pass_level.body': 'You progress on the season pass.',
    'notif.sync_error.title': 'Sync error',
    'notif.sync_error.body': '{message}',
    'notif.personal_record.title': 'Personal record',
    'notif.personal_record.body': 'New record on {metric_label}: {value}.',
    'notif.threshold_crossed.title': 'Threshold crossed',
    'notif.threshold_crossed.body': 'You crossed a {metric_label} threshold: {value}.',
    'notif.trend_consolidate.title': 'Focus to consolidate',
    'notif.trend_consolidate.body': 'One of your performance areas has been dipping lately — a chance to shore it up.',
    'notif.friend_added.title': '{gamertag} added to your friends',
    'notif.friend_added.body': 'Shared sessions will be reclassified as squad in the background.',
    'notif.friend_sync_completed.title': 'Friend sessions updated',
    'notif.friend_sync_completed.body': '{promoted} match(es) reclassified as squad-friends.',
    'notif.test.title': 'Test notification',
    'notif.test.body': 'The notifications pipeline is working correctly.',
    'notif.data_health_warning.title': 'Database audit: {warnings_total} anomaly(ies) found',
    'notif.data_health_warning.body': '{uuids_raw} raw UUID(s), {lying_bits_events} lying bit(s), {garbage_banner_urls} stale URL(s). {hint}',
    'notif.career_rank.title': 'Rank {rank} reached',
    'notif.career_rank.body': 'You unlocked {rank_name} (from rank {previous}).',
    'notif.skill_tier.title': 'New tier {tier} ({rating_type})',
    'notif.skill_tier.body': 'Playlist {playlist_group} — {tier} {sub_tier} (was: {previous_tier} {previous_sub_tier}).',
    'notif.battlepass_completed.title': 'Battle pass completed',
    'notif.battlepass_completed.body': '{count} BP track(s) reached max rank — congrats!',
    'notif.citation_tier.title': 'New commendation tier',
    'notif.citation_tier.body': '{count} tier(s) crossed since last sync.',
    'notif.citation_mastery.title': 'Commendation mastered',
    'notif.citation_mastery.body': '{count} commendation(s) at 100% — nice!',
    // ─── Progression V2 (Ascension) — proactive coach ────────────────────
    'notif.record_near_miss.title': 'Approaching a record',
    'notif.record_near_miss.body': 'Your {metric_label} over {period_label} is approaching your PB ({value} vs {target}).',
    'notif.milestone_unlocked.title': 'Milestone unlocked: {title_en}',
    'notif.milestone_unlocked.body': 'You just unlocked « {title_en} » — congrats!',
    'notif.milestone_near_miss.title': 'Almost there on a milestone',
    'notif.milestone_near_miss.body': 'Just a few more steps to unlock « {title_en} ».',
    'notif.lusr_tier_approach.title': '{gap} pts from {next_tier_name}',
    'notif.lusr_tier_approach.body': 'Your LUSR rating is within reach of the next sub-tier {next_tier_name}.',
    'notif.streak_milestone.title': '{length}-day streak!',
    'notif.streak_milestone.body': 'You reached the {length}-day milestone — PP multiplier ×{multiplier}.',
    'notif.comeback_welcome.title': 'Welcome back!',
    'notif.comeback_welcome.body': 'You returned after {days_away} days away — your streak shield is ready.',
    'notif.title_ready.title': '{title_name} is ready',
    'notif.title_ready.body': 'Your {title_name} data is synced — explore your stats.',
    'notif.rival_encounter.title': 'Rival encountered',
    'notif.rival_encounter.body': 'You crossed paths with {gamertag} again: {kills} frags / {deaths} deaths.',
    'notif.medal_first_earned.title': 'First-time medal',
    'notif.medal_first_earned.body': 'You earned {medal_name_en} for the first time — nice!',
    'notif.medal_first_earned.recap.title': 'First-time medals',
    'notif.medal_first_earned.recap.body': '{count} medals earned for the first time.',
  },

  relJustNow: 'just now',
  relMinutesAgo: (n) => (n <= 1 ? '1 min ago' : `${n} min ago`),
  relHoursAgo: (n) => (n <= 1 ? '1 h ago' : `${n} h ago`),
  relDaysAgo: (n) => (n <= 1 ? '1 d ago' : `${n} d ago`),
  relOnDate: (iso) => new Date(iso).toLocaleDateString('en-US'),
}

const DICTS: Record<Locale, NotificationsText> = { en: EN }

export function getNotificationsText(locale: Locale): NotificationsText {
  return DICTS[locale] ?? EN
}
