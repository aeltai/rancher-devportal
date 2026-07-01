<template>
  <img
    v-if="src"
    :src="src"
    :class="['dp-offering-icon-img', { 'dp-offering-icon-img--large': large }]"
    alt=""
  >
  <i
    v-else
    :class="['icon', iconClass]"
  />
</template>

<script>
import { offeringIconClass, offeringIconLarge, offeringIconSrc } from './offeringIcons';

export default {
  name: 'OfferingIcon',
  props: {
    offering: { type: Object, default: null },
  },
  computed: {
    src() {
      return offeringIconSrc(this.offering);
    },
    iconClass() {
      return offeringIconClass(this.offering);
    },
    large() {
      return offeringIconLarge(this.offering);
    },
  },
};
</script>

<style scoped>
.dp-offering-icon-img {
  width: 1.45em;
  height: 1.45em;
  object-fit: contain;
  flex-shrink: 0;
}

.dp-offering-icon-img--large {
  width: 52px;
  height: 52px;
  margin: 2px 0 4px;
}
</style>
