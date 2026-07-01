import { PORTAL_PRODUCT_ICON } from './offeringIcons';

const BLANK_CLUSTER = '_';

export function init($plugin, store) {
  const PRODUCT = 'platform';
  const PORTAL_PAGE = 'portal';

  const { product, basicType, virtualType } = $plugin.DSL(store, PRODUCT);

  product({
    svg: PORTAL_PRODUCT_ICON,
    label: 'Geeko-Ops',
    inStore: 'management',
    weight: 90,
    to: {
      name: `${PRODUCT}-c-cluster-${PORTAL_PAGE}`,
      params: { product: PRODUCT, cluster: BLANK_CLUSTER },
    },
  });

  virtualType({
    label: 'Geeko-Ops',
    name: PORTAL_PAGE,
    route: {
      name: `${PRODUCT}-c-cluster-${PORTAL_PAGE}`,
      params: { product: PRODUCT, cluster: BLANK_CLUSTER },
    },
    icon: 'marketplace',
  });

  basicType([PORTAL_PAGE]);
}
