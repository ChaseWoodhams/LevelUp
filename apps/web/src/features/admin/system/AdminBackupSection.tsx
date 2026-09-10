/**
 * AdminBackupSection — statut + déclenchement de la sauvegarde restic des bases
 * DuckDB, rapatrié depuis l'onglet « Sauvegarde » des Réglages utilisateur.
 *
 * Le backup est une opération d'instance (toutes les bases, rétention gérée au
 * niveau serveur), pas une préférence utilisateur : sa place est la page
 * Admin · Système. Composant autonome qui câble son propre i18n — même pattern
 * que AdminSyncSettingsSection.
 */
import { getSettingsText } from '@/features/settings/i18n'
import { BackupTab } from '@/features/settings/BackupTab'
import { SectionHeader } from '../components/SectionHeader'

export function AdminBackupSection() {
  const t = getSettingsText()

  return (
    <section className="space-y-3">
      <SectionHeader title={t.tabBackup} />
      <BackupTab t={t} frozen={false} />
    </section>
  )
}
