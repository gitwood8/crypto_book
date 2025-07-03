package types

type ConfirmationTemplateType struct {
	MessageText     string
	ConfirmText     string
	ConfirmCallback string
	CancelText      string
	CancelCallback  string
	NextState       string
}

var ConfirmationTemplates = map[string]ConfirmationTemplateType{
	"rename_portfolio": {
		MessageText:     "Are you sure you want to rename portfolio *'%s'* to *'%s'*?",
		ConfirmText:     "Yes, rename",
		ConfirmCallback: "confirm_portfolio_rename",
		CancelText:      "Cancel",
		CancelCallback:  "cancel_action",
		NextState:       "waiting_rename_portfolio_decision",
	},
	"delete_portfolio": {
		MessageText:     "Are you sure? This will permanently delete the portfolio *'%s'* and its transactions.",
		ConfirmText:     "Yes, delete",
		ConfirmCallback: "confirm_portfolio_deletion",
		CancelText:      "Cancel",
		CancelCallback:  "cancel_action",
		NextState:       "waiting_delete_portfolio_decision",
	},
	"change_default_portfolio": {
		MessageText:     "Are you sure you want to set *'%s'* as *default* portfolio?",
		ConfirmText:     "Yes, change default",
		ConfirmCallback: "confirm_portfolio_change_default",
		CancelText:      "Cancel",
		CancelCallback:  "cancel_action",
		// NextState:       "waiting_delete_portfolio_decision",
	},
}

type Actiontype struct {
	TgText       string
	CallBackName string
}

type PortfolioAsset struct {
	Pair        string
	TotalAmount float64
	TotalUSD    float64
}

type PortfolioSummary struct {
	Name   string
	Assets []PortfolioAsset
}

func (e *PriceDataError) Error() string {
	return e.Message
}

// Additional types for price data fetching
type PriceDataError struct {
	InvalidPairs []string // Pairs that couldn't be fetched
	Message      string   // Error message
}

type BinanceErrorResponse struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
}
