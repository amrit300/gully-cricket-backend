func createUser(c *fiber.Ctx) error {

	var req RegisterRequest

	if err := c.BodyParser(&req); err != nil {

		log.Println("BODY PARSE ERROR:", err)

		return c.Status(400).JSON(fiber.Map{
			"error": "invalid request",
		})
	}

	log.Println("USERNAME:", req.Username)
	log.Println("INIT DATA:", req.InitData)

	botToken := os.Getenv("TELEGRAM_BOT_TOKEN")

	if botToken == "" {
		log.Println("BOT TOKEN MISSING")
		return c.Status(500).JSON(fiber.Map{
			"error": "server configuration error",
		})
	}

	/*
	Verify Telegram HMAC signature
	*/

	if !verifyTelegram(req.InitData, botToken) {

		log.Println("TELEGRAM VERIFICATION FAILED")

		return c.Status(403).JSON(fiber.Map{
			"error": "telegram verification failed",
		})
	}

	/*
	Extract Telegram user id
	*/

	values, err := url.ParseQuery(req.InitData)

	if err != nil {
		log.Println("INIT DATA PARSE ERROR:", err)
		return c.Status(400).JSON(fiber.Map{
			"error": "invalid telegram payload",
		})
	}

	userJSON := values.Get("user")

if userJSON == "" {

	log.Println("USER FIELD MISSING IN INIT DATA")

	return c.Status(400).JSON(fiber.Map{
		"error": "telegram user missing",
	})
}

var telegramUser struct {
	ID        int    `json:"id"`
	Username  string `json:"username"`
	FirstName string `json:"first_name"`
}

if err := json.Unmarshal([]byte(userJSON), &telegramUser); err != nil {

	log.Println("USER JSON PARSE ERROR:", err)

	return c.Status(400).JSON(fiber.Map{
		"error": "invalid telegram user",
	})
}

	log.Println("TELEGRAM USER ID:", telegramUser.ID)

	query := `
	INSERT INTO users (username, telegram)
	VALUES ($1,$2)
	ON CONFLICT (telegram)
	DO UPDATE SET username=EXCLUDED.username
	RETURNING id
	`

	var id int

err = db.QueryRow(
	query,
	req.Username,
	telegramUser.ID,
).Scan(&id)

if err != nil {

	log.Printf("USER INSERT ERROR: %+v\n", err)

	// 🔥 RETURN REAL ERROR TO FRONTEND
	return c.Status(500).JSON(fiber.Map{
		"error": err.Error(),
	})

}
	log.Println("USER CREATED:", id)

	return c.JSON(fiber.Map{
		"user_id": id,
	})
}
