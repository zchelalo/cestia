import { useTranslation } from "react-i18next";
import styles from "@/pages/Page.module.css";
import { formatDate } from "@/shared/money";
import type { Activity } from "@/shared/types";

const labels: Record<string, string> = {
	"list.created": "Lista creada",
	"list.updated": "Lista actualizada",
	"list.archived": "Lista archivada",
	"list.restored": "Lista restaurada",
	"item.created": "Artículo agregado",
	"item.updated": "Artículo actualizado",
	"item.deleted": "Artículo eliminado",
	"purchase.recorded": "Compra registrada",
	"purchase.corrected": "Compra corregida",
	"purchase.reversed": "Compra revertida",
	"group.created": "Grupo creado",
	"group.updated": "Grupo actualizado",
	"group.archived": "Grupo archivado",
	"group.restored": "Grupo restaurado",
	"group.owner_transferred": "Propiedad transferida",
	"member.role_changed": "Rol actualizado",
	"member.removed": "Integrante eliminado",
	"member.left": "Integrante salió",
	"invitation.created": "Invitación creada",
	"invitation.resent": "Invitación reenviada",
	"invitation.canceled": "Invitación cancelada",
	"invitation.accepted": "Invitación aceptada",
};

export function ActivityTimeline({ events }: { events: Activity[] }) {
	const { t } = useTranslation();
	if (events.length === 0)
		return <div className={styles.empty}>{t("common.noResults")}</div>;
	return (
		<div className={styles.timeline}>
			{events.map((event) => (
				<div key={event.id} className={styles.timelineItem}>
					<span className={styles.dot} aria-hidden="true" />
					<div>
						<div className={styles.rowTitle}>
							{labels[event.action] ?? event.action}
						</div>
						<div className={styles.eyebrowInline}>
							{event.actor_name ? `${event.actor_name} · ` : ""}
							{formatDate(event.created_at, true)}
						</div>
					</div>
				</div>
			))}
		</div>
	);
}
