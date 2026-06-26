const BLANK_CLUSTER = '_';

export function init($plugin, store) {
  const PRODUCT = 'platform';
  const PORTAL_PAGE = 'portal';

  const { product, basicType, virtualType } = $plugin.DSL(store, PRODUCT);

  product({
    icon: 'globe',
    label: 'Developer Portal',
    inStore: 'management',
    weight: 90,
    to: {
      name: `${PRODUCT}-c-cluster-${PORTAL_PAGE}`,
      params: { product: PRODUCT, cluster: BLANK_CLUSTER },
    },
  });

  virtualType({
    label: 'Developer Portal',
    name: PORTAL_PAGE,
    route: {
      name: `${PRODUCT}-c-cluster-${PORTAL_PAGE}`,
      params: { product: PRODUCT, cluster: BLANK_CLUSTER },
    },
    icon: 'globe',
  });

  basicType([PORTAL_PAGE]);
}
