package telegram_bot

type ConfirmationTemplate struct {
	MessageText     string
	ConfirmText     string
	ConfirmCallback string
	CancelText      string
	CancelCallback  string
	NextState       string
}

var confirmationTemplates = map[string]ConfirmationTemplate{
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

type Action struct {
	TgText       string
	CallBackName string
}
