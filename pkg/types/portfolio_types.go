package types

const ServiceDescription = `*🤖 Wood Post - Crypto Portfolio Tracker*

*What I do:*
📈 I help you track your cryptocurrency investments and transactions across multiple portfolios.

*Key Features:*
💼 *Portfolio Management*
• Create up to 2 portfolios (free tier)
• Set default portfolio for quick access
• Rename and manage your portfolios

💰 *Transaction Tracking*
• Record BUY/SELL transactions
• Support for all major crypto pairs (BTCUSDT, ETHUSDT, etc.)
• Automatic USD value calculation
• View your last 5 transactions with beautiful formatting

📊 *Smart Features*
• Remembers your most-used trading pairs
• Quick date selection (Today, Yesterday, etc.)
• Input validation to prevent errors
• Clean, emoji-rich interface

*How it works:*
1️⃣ Start by creating your first portfolio
2️⃣ Add transactions with amount, price, and date
3️⃣ View your transaction history anytime
4️⃣ Track your crypto investments easily

*Getting Started:*
Just type /start and I'll guide you through creating your first portfolio and adding transactions!

*Note:* This is a personal tracking tool. Your data stays private and secure. 🔒`

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
