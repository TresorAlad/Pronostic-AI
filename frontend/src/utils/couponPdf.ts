import jsPDF from 'jspdf';
import autoTable from 'jspdf-autotable';
import type { CouponSelection } from '../api';
import { marketCategory, marketLabel } from './marketLabels';

export interface CouponPdfInput {
  id: string;
  name?: string;
  userName?: string;
  selections: CouponSelection[];
  combinedOdd?: number;
  disclaimer?: string;
}

const BRAND = { r: 16, g: 185, b: 129 };
const DARK = { r: 20, g: 27, b: 45 };

function formatOdd(n: number) {
  return n.toFixed(2).replace('.', ',');
}

function resolveLabel(sel: CouponSelection) {
  return sel.market_label ?? marketLabel(sel.market || sel.selection);
}

export function exportCouponPdf(coupon: CouponPdfInput) {
  const doc = new jsPDF({ unit: 'mm', format: 'a4' });
  const pageW = doc.internal.pageSize.getWidth();
  const now = new Date().toLocaleString('fr-FR');

  doc.setFillColor(DARK.r, DARK.g, DARK.b);
  doc.rect(0, 0, pageW, 42, 'F');

  doc.setFillColor(BRAND.r, BRAND.g, BRAND.b);
  doc.circle(18, 21, 8, 'F');
  doc.setTextColor(255, 255, 255);
  doc.setFontSize(10);
  doc.setFont('helvetica', 'bold');
  doc.text('PA', 15.2, 22.5);

  doc.setFontSize(18);
  doc.text('Pronostic-AI', 30, 18);
  doc.setFontSize(10);
  doc.setFont('helvetica', 'normal');
  doc.text(coupon.name ?? 'Coupon IA', 30, 26);
  doc.text(`Généré le ${now}`, 30, 32);
  if (coupon.userName) {
    doc.text(`Par ${coupon.userName}`, pageW - 14, 32, { align: 'right' });
  }

  doc.setTextColor(40, 40, 40);
  doc.setFontSize(11);
  doc.text('Total buts · BTTS · Buts par équipe', 14, 50);

  const combined =
    coupon.combinedOdd ??
    coupon.selections.reduce((acc, s) => acc * (s.bookmaker_odd ?? 1 / Math.max(s.confidence, 0.01)), 1);

  doc.setFont('helvetica', 'bold');
  doc.text(`Cote combinée : ${formatOdd(combined)} / 50 max`, 14, 58);
  doc.setFont('helvetica', 'normal');

  const rows = coupon.selections.map((sel, i) => [
    String(i + 1),
    `${sel.home_team} vs ${sel.away_team}`,
    sel.league_name ?? '-',
    sel.market_category ?? marketCategory(sel.market),
    resolveLabel(sel),
    `${Math.round(sel.confidence * 100)} %`,
    sel.bookmaker_odd ? formatOdd(sel.bookmaker_odd) : '-',
  ]);

  autoTable(doc, {
    startY: 64,
    head: [['#', 'Match', 'Ligue', 'Catégorie', 'Sélection', 'Confiance', 'Cote']],
    body: rows,
    styles: { fontSize: 9, cellPadding: 2.5 },
    headStyles: { fillColor: [BRAND.r, BRAND.g, BRAND.b], textColor: 255 },
    alternateRowStyles: { fillColor: [245, 247, 250] },
    columnStyles: {
      0: { cellWidth: 8 },
      5: { halign: 'center' },
      6: { halign: 'right' },
    },
  });

  const finalY = (doc as jsPDF & { lastAutoTable?: { finalY: number } }).lastAutoTable?.finalY ?? 200;
  doc.setFontSize(8);
  doc.setTextColor(100, 100, 100);
  const disclaimer =
    coupon.disclaimer ??
    'Estimations statistiques, aucune garantie de gain. Jouer comporte des risques.';
  doc.text(disclaimer, 14, finalY + 10, { maxWidth: pageW - 28 });
  doc.text(`Réf. coupon ${coupon.id}`, 14, finalY + 20);

  doc.save(`coupon-prono-${coupon.id.slice(0, 8)}.pdf`);
}
