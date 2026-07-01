import geekoJacketIcon from './assets/geeko-jacket-icon.png';
import geekoSidebar from './assets/geeko-sidebar.svg';

const CUSTOM_OFFERING_ICONS = {
  'geeko-drugs': geekoJacketIcon,
};

const OFFERING_ICON_CLASSES = {
  namespace: 'icon-namespace',
  cluster: 'icon-cluster',
  helm: 'icon-helm',
  crd: 'icon-crd',
  generic: 'icon-file',
  copy: 'icon-copy',
  vm: 'icon-vm',
  apps: 'icon-apps',
  file: 'icon-file',
  fleet: 'icon-fleet',
};

/** Sidebar app-bar icon — monochrome SVG traced from mascot (Shell recolors via CSS filter). */
export const PORTAL_PRODUCT_ICON = geekoSidebar;

export function offeringIconSrc(offering) {
  if (!offering) return null;
  return CUSTOM_OFFERING_ICONS[offering.icon] || CUSTOM_OFFERING_ICONS[offering.id] || null;
}

export function offeringIconClass(offering) {
  if (!offering) return 'icon-apps';
  const map = OFFERING_ICON_CLASSES;
  return map[offering.icon] || map[offering.kind] || 'icon-apps';
}

export function offeringIconLarge(offering) {
  return !!(offering && (offering.icon === 'geeko-drugs' || offering.id === 'geeko-drugs'));
}
