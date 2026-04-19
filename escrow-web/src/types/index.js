/**
 * @typedef {'PAYING' | 'PENDING' | 'IN_PROGRESS' | 'COMPLETED' | 'FAILED' | 'CANCELED'} BountyStatus
 * @typedef {'APPLIED' | 'ACCEPTED' | 'REJECTED'} ApplicationStatus
 * @typedef {'PENDING' | 'SUCCESS' | 'FAILED'} PaymentStatus
 * @typedef {'PROCESSING' | 'SUCCESS' | 'FAILED'} WithdrawalStatus
 */

/**
 * @typedef {Object} Bounty
 * @property {number} id
 * @property {string} employer_username
 * @property {number} employer_account_id
 * @property {string} title
 * @property {string} description
 * @property {number} reward_amount 分（cents）
 * @property {BountyStatus} status
 * @property {string} created_at
 * @property {string} updated_at
 * @property {BountyApplication[]} [applications]
 */

/**
 * @typedef {Object} BountyApplication
 * @property {number} id
 * @property {number} bounty_id
 * @property {string} hunter_username
 * @property {number} hunter_account_id
 * @property {ApplicationStatus} status
 * @property {string} created_at
 */

/**
 * @typedef {Object} UserProfile
 * @property {string} username
 * @property {string} full_name
 * @property {string} avatar_url
 * @property {string} bio
 * @property {number} completed_count
 * @property {number} posted_count
 * @property {number} total_earnings
 * @property {number} rating
 */

/**
 * @typedef {Object} Account
 * @property {number} id
 * @property {string} owner
 * @property {number} balance 分（cents）
 * @property {string} currency
 */

/**
 * @typedef {Object} Payment
 * @property {number} id
 * @property {string} username
 * @property {number} account_id
 * @property {number} amount 分（cents）
 * @property {string} currency
 * @property {PaymentStatus} status
 * @property {string} alipay_trade_no
 */

/**
 * @typedef {Object} Withdrawal
 * @property {number} id
 * @property {string} username
 * @property {number} account_id
 * @property {number} amount 分（cents）
 * @property {string} alipay_account
 * @property {string} alipay_real_name
 * @property {WithdrawalStatus} status
 * @property {string} error_msg
 */

/**
 * @typedef {Object} LoginResponse
 * @property {string} access_token
 * @property {string} refresh_token
 */

/**
 * @typedef {Object} ApiError
 * @property {string} message
 * @property {number} [status]
 */
