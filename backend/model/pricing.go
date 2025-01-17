package model

type PricingProductType string

const (
	PricingProductTypeBase     PricingProductType = "BASE"
	PricingProductTypeMembers  PricingProductType = "MEMBERS"
	PricingProductTypeSessions PricingProductType = "SESSIONS"
	PricingProductTypeErrors   PricingProductType = "ERRORS"
	PricingProductTypeLogs     PricingProductType = "LOGS"
)

type PricingSubscriptionInterval string

const (
	PricingSubscriptionIntervalMonthly PricingSubscriptionInterval = "MONTHLY"
	PricingSubscriptionIntervalAnnual  PricingSubscriptionInterval = "ANNUAL"
)
