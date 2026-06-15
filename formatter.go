package main

import (
	"fmt"
	"log"
	"strings"
	"time"
)

func fmtEmoji(meal Meal) string {
	emojis := ""
	for _, icon := range meal.Icons {
		if emoji, exists := iconsMap[icon]; exists {
			emojis += " " + emoji
		}
	}
	return meal.Name + emojis
}

func fmtMeal(meals []Meal) string {
	var formatted []string
	for _, meal := range meals {
		line := fmtEmoji(meal)
		if meal.Changed {
			line += " (alterado)"
		}
		formatted = append(formatted, line)
	}
	return strings.Join(formatted, "\n")
}

func fmtMenu(evt EventLambda) string {
	var sections []string
	layout := "02/01/2006"
	menu := evt.ResponsePayload

	log.Printf("Formatting menu: %+v", menu)

	t, err := time.Parse(layout, menu.Date)
	if err != nil {
		fmt.Println("Error parsing date:", err)
	}

	weekday := t.Weekday()
	titleStr := fmt.Sprintf(title, strings.ToUpper(menu.Restaurant.Name), weekDay[int(weekday)], menu.Date)

	sections = append(sections, titleStr)

	for _, mealServed := range mealOrder {
		if meals, exists := menu.Meals[mealServed]; exists {
			header := mealTypeHeaders[mealServed]
			mealSection := header + "\n" + fmtMeal(meals)
			sections = append(sections, mealSection)
		}
	}

	sections = append(sections, legend, fmt.Sprintf(taken, menu.Restaurant.Url), fmt.Sprintf(channel, evt.RequestPayload.WhatsAppLink), copyright)
	return strings.Join(sections, "\n\n")
}
