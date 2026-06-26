import DevPortalPage from '../DevPortalPage.vue';

const PRODUCT = 'platform';
const PORTAL_PAGE = 'portal';
const BLANK_CLUSTER = '_';

const routes = [
  {
    name: `${PRODUCT}-c-cluster-${PORTAL_PAGE}`,
    path: `/${PRODUCT}/c/:cluster/${PORTAL_PAGE}`,
    component: DevPortalPage,
    meta: {
      product: PRODUCT,
      cluster: BLANK_CLUSTER,
    },
  },
];

export default routes;
