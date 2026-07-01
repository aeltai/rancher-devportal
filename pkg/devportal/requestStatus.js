export const REQUEST_STATUS_TABS = [
  { id: 'all', label: 'All' },
  { id: 'new', label: 'New' },
  { id: 'approved', label: 'Approved' },
  { id: 'declined', label: 'Declined' },
];

export const USER_STATUS_TABS = [
  { id: 'all', label: 'All' },
  { id: 'new', label: 'Awaiting approval' },
  { id: 'approved', label: 'Active' },
  { id: 'declined', label: 'Declined' },
];

export function statusTabsForVariant(variant) {
  return variant === 'user' ? USER_STATUS_TABS : REQUEST_STATUS_TABS;
}

export function requestStatusBucket(r, needsAdminApproval) {
  if (!r) return 'approved';
  if (r.phase === 'Rejected' || r.phase === 'Failed') return 'declined';
  if (needsAdminApproval(r) || r.phase === 'PendingApproval') return 'new';
  return 'approved';
}

export function filterRequestsByStatus(requests, tab, needsAdminApproval) {
  if (!tab || tab === 'all') return requests;
  return requests.filter((r) => requestStatusBucket(r, needsAdminApproval) === tab);
}

export function countRequestsByStatus(requests, needsAdminApproval) {
  const counts = { all: requests.length, new: 0, approved: 0, declined: 0 };
  requests.forEach((r) => {
    const bucket = requestStatusBucket(r, needsAdminApproval);
    counts[bucket] += 1;
  });
  return counts;
}

export function statusTabEmptyMessage(tab, variant = 'admin') {
  const adminMessages = {
    all: 'No platform requests yet.',
    new: 'No new requests awaiting review.',
    approved: 'No approved or in-progress requests.',
    declined: 'No declined or failed requests.',
  };
  const userMessages = {
    all: 'You have not submitted any requests yet.',
    new: 'None of your requests are waiting for admin approval.',
    approved: 'You have no active or ready environments.',
    declined: 'You have no declined or failed requests.',
  };
  const messages = variant === 'user' ? userMessages : adminMessages;
  return messages[tab] || messages.all;
}
