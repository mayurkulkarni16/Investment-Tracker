export function normalizeBank(value) {
  const bank = value.toLowerCase().replace(/[^a-z0-9]/g, '');
  if (/^hdfc(?:bank|bk)?(?:ltd|limited)?$/.test(bank)) return 'hdfc';
  if (/^icici(?:bank|b)?(?:ltd|limited)?$/.test(bank)) return 'icici';
  if (/^axis(?:bank|bk)?(?:ltd|limited)?$/.test(bank)) return 'axis';
  if (/^(?:sbi(?:bank|card|cards)?|statebankofindia)(?:ltd|limited)?$/.test(bank)) return 'sbi';
  if (/^yes(?:bank|bnk)?(?:ltd|limited)?$/.test(bank)) return 'yesbank';
  if (/^(?:onecard|onecrd|oncrd)$/.test(bank)) return 'onecard';
  return bank;
}

export function accountKey(account) {
  return `${account.type}:${account.id}`;
}

export function memberForAccount(account, deviceMemberId) {
  if (account.member_ids.length === 1) return account.member_ids[0];
  if (account.member_ids.length > 1) {
    return account.member_ids.includes(deviceMemberId) ? deviceMemberId : '';
  }
  return deviceMemberId;
}

export function suggestAttribution(draft, context, deviceMemberId) {
  const phoneMember = context.members.some(member => member.id === deviceMemberId) ? deviceMemberId : '';
  if (!/^\d{4}$/.test(draft.account)) {
    return { account: null, profileId: phoneMember, message: 'No account or card last four was found. Choose an account and review the member.' };
  }

  const bank = normalizeBank(draft.bank);
  const candidates = context.accounts.filter(account => {
    if (account.last_four !== draft.account) return false;
    const savedBank = normalizeBank(account.bank);
    return !bank || !savedBank || bank === savedBank;
  });
  if (candidates.length !== 1) {
    return {
      account: null,
      profileId: phoneMember,
      message: candidates.length
        ? 'Several saved accounts or cards match these digits. Choose the correct one and review the member.'
        : 'No saved account or card matches these bank details. Link one and review the member.',
    };
  }

  const account = candidates[0];
  const profileId = memberForAccount(account, phoneMember);
  let message = `Matched ${account.name || 'a saved account'}; check the match before confirming.`;
  if (!account.bank || !draft.bank) message += ' The bank could not be verified.';
  if (account.member_ids.length > 1) {
    message += profileId
      ? ' This is a joint account; the device member was selected. Check who made the transaction.'
      : ' This is a joint account; choose who made the transaction.';
  } else if (account.member_ids.length === 0) {
    message += ' This account has no family owner assigned.';
  }
  if (account.status === 'closed') message += ' This account is marked closed.';
  return { account, profileId, message };
}

function duplicateReason(entry, candidate) {
  if (entry.id && candidate.id === entry.id) return null;
  if (entry.source_key && entry.source_key === candidate.source_key) return 'exact';
  if (candidate.status === 'rejected') return null;
  if (!/^\d{4}$/.test(entry.account) || entry.account !== candidate.account
    || entry.date !== candidate.date || entry.kind !== candidate.kind
    || entry.amount_paise !== candidate.amount_paise) return null;
  if (entry.linked_account_id && candidate.linked_account_id
    && (entry.linked_account_id !== candidate.linked_account_id
      || entry.linked_account_type !== candidate.linked_account_type)) return null;
  const bank = normalizeBank(entry.bank);
  const otherBank = normalizeBank(candidate.bank);
  if (bank && otherBank && bank !== otherBank) return null;

  if (entry.sms_received_at && candidate.sms_received_at) {
    const time = Date.parse(entry.sms_received_at);
    const otherTime = Date.parse(candidate.sms_received_at);
    if (Number.isFinite(time) && Number.isFinite(otherTime)) {
      return Math.abs(time - otherTime) <= 5 * 60 * 1000 ? 'within_minutes' : null;
    }
  }
  return 'date_only';
}

export function findDuplicates(entry, entries) {
  return entries.flatMap(candidate => {
    const reason = duplicateReason(entry, candidate);
    return reason ? [{ entry: candidate, reason }] : [];
  });
}

export function indexDuplicates(entries) {
  const buckets = new Map();
  const result = {};
  for (const entry of entries) {
    if (entry.status === 'rejected' || !/^\d{4}$/.test(entry.account)) continue;
    const key = `${entry.date}|${entry.kind}|${entry.amount_paise}|${entry.account}`;
    const bucket = buckets.get(key) || [];
    for (const other of bucket) {
      const reason = duplicateReason(entry, other);
      if (!reason) continue;
      (result[entry.id] ||= []).push({ entry: other, reason });
      (result[other.id] ||= []).push({ entry, reason });
    }
    bucket.push(entry);
    buckets.set(key, bucket);
  }
  return result;
}
