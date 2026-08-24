package main

import (
	"context"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type templateSeed struct {
	slug         string
	name         string
	description  string
	sortOrder    int
	defaultTheme string
}

func seedTemplates(ctx context.Context, db *sqlx.DB) error {
	templates := []templateSeed{
		{
			slug: "minimal", name: "Minimal", sortOrder: 1,
			description: "Fondo blanco, tipografía limpia, botones simples.",
			defaultTheme: `{"background_type":"color","background_value":"#ffffff","primary_color":"#111827",
				"secondary_color":"#111827","text_color":"#111827","button_text_color":"#ffffff",
				"logo_background_color":"#111827","logo_text_color":"#ffffff","logo_display_mode":"initial","logo_shape":"circle",
				"font_family":"Inter","button_style":"rounded","button_shadow":false}`,
		},
		{
			slug: "business", name: "Business", sortOrder: 2,
			description: "Paleta corporativa en azul, botones sólidos.",
			defaultTheme: `{"background_type":"color","background_value":"#f8fafc","primary_color":"#1e3a8a",
				"secondary_color":"#1e3a8a","text_color":"#0f172a","button_text_color":"#ffffff",
				"logo_background_color":"#1e3a8a","logo_text_color":"#ffffff","logo_display_mode":"initial","logo_shape":"square",
				"font_family":"Inter","button_style":"square","button_shadow":true}`,
		},
		{
			slug: "restaurant", name: "Restaurant", sortOrder: 3,
			description: "Tonos cálidos, ideal para menús y gastronomía.",
			defaultTheme: `{"background_type":"gradient","background_value":"linear-gradient(160deg,#7c2d12,#b91c1c)",
				"primary_color":"#fef3c7","secondary_color":"#fef3c7","text_color":"#fff7ed","button_text_color":"#451a03",
				"logo_background_color":"#fef3c7","logo_text_color":"#451a03","logo_display_mode":"initial","logo_shape":"circle",
				"font_family":"Poppins","button_style":"pill","button_shadow":true}`,
		},
		{
			slug: "modern", name: "Modern", sortOrder: 4,
			description: "Gradiente vibrante, botones con sombra.",
			defaultTheme: `{"background_type":"gradient","background_value":"linear-gradient(160deg,#0f172a,#4338ca)",
				"primary_color":"#ffffff","secondary_color":"#ffffff","text_color":"#f8fafc","button_text_color":"#1e1b4b",
				"logo_background_color":"#ffffff","logo_text_color":"#1e1b4b","logo_display_mode":"initial","logo_shape":"circle",
				"font_family":"Poppins","button_style":"rounded","button_shadow":true}`,
		},
		{
			slug: "elegant", name: "Elegant", sortOrder: 5,
			description: "Tonos neutros y tipografía serif para negocios premium.",
			defaultTheme: `{"background_type":"color","background_value":"#f5f5f4","primary_color":"#292524",
				"secondary_color":"#292524","text_color":"#292524","button_text_color":"#f5f5f4",
				"logo_background_color":"#292524","logo_text_color":"#f5f5f4","logo_display_mode":"initial","logo_shape":"circle",
				"font_family":"Playfair Display","button_style":"outline","button_shadow":false}`,
		},
		{
			slug: "dark", name: "Dark", sortOrder: 6,
			description: "Modo oscuro con acentos vibrantes.",
			defaultTheme: `{"background_type":"color","background_value":"#0b0f19","primary_color":"#22d3ee",
				"secondary_color":"#22d3ee","text_color":"#e2e8f0","button_text_color":"#04222b",
				"logo_background_color":"#22d3ee","logo_text_color":"#04222b","logo_display_mode":"initial","logo_shape":"circle",
				"font_family":"Inter","button_style":"rounded","button_shadow":true}`,
		},
		{
			slug: "colorful", name: "Colorful", sortOrder: 7,
			description: "Colores llamativos y botones tipo píldora.",
			defaultTheme: `{"background_type":"gradient","background_value":"linear-gradient(160deg,#f472b6,#facc15)",
				"primary_color":"#ffffff","secondary_color":"#ffffff","text_color":"#1f2937","button_text_color":"#831843",
				"logo_background_color":"#ffffff","logo_text_color":"#831843","logo_display_mode":"initial","logo_shape":"circle",
				"font_family":"Poppins","button_style":"pill","button_shadow":false}`,
		},
		{
			slug: "nature", name: "Nature", sortOrder: 8,
			description: "Verdes tierra, ideal para negocios orgánicos, wellness o cafeterías.",
			defaultTheme: `{"background_type":"gradient","background_value":"linear-gradient(160deg,#f0fdf4,#dcfce7)",
				"primary_color":"#166534","secondary_color":"#166534","text_color":"#14532d","button_text_color":"#f0fdf4",
				"logo_background_color":"#166534","logo_text_color":"#f0fdf4","logo_display_mode":"initial","logo_shape":"rounded",
				"font_family":"Nunito","button_style":"rounded","button_shadow":false}`,
		},
		{
			slug: "monochrome", name: "Monochrome", sortOrder: 9,
			description: "Blanco y negro puro, muy editorial, para moda y fotografía.",
			defaultTheme: `{"background_type":"color","background_value":"#ffffff","primary_color":"#000000",
				"secondary_color":"#000000","text_color":"#000000","button_text_color":"#ffffff",
				"logo_background_color":"#000000","logo_text_color":"#ffffff","logo_display_mode":"initial","logo_shape":"square",
				"font_family":"Montserrat","button_style":"square","button_shadow":false}`,
		},
		{
			slug: "sunset-vibes", name: "Sunset Vibes", sortOrder: 10,
			description: "Degradado naranja-morado, energético, para vida nocturna y eventos.",
			defaultTheme: `{"background_type":"gradient","background_value":"linear-gradient(160deg,#f97316,#7c3aed)",
				"primary_color":"#ffffff","secondary_color":"#111827","text_color":"#fff7ed","button_text_color":"#ffffff",
				"logo_background_color":"#111827","logo_text_color":"#ffffff","logo_display_mode":"initial","logo_shape":"circle",
				"font_family":"Poppins","button_style":"pill","button_shadow":true}`,
		},
	}

	for _, t := range templates {
		var id string
		err := db.GetContext(ctx, &id, `SELECT id FROM templates WHERE slug = ?`, t.slug)
		if err == nil {
			// Template already exists — refresh its theme/description in place so
			// re-running the seed after a template tweak actually applies it.
			_, err := db.ExecContext(ctx, `
				UPDATE templates SET name = ?, description = ?, default_theme = ?, sort_order = ?
				WHERE id = ?`,
				t.name, t.description, t.defaultTheme, t.sortOrder, id,
			)
			if err != nil {
				return err
			}
			continue
		}

		_, err = db.ExecContext(ctx, `
			INSERT INTO templates (id, slug, name, description, default_theme, is_active, sort_order)
			VALUES (?, ?, ?, ?, ?, 1, ?)`,
			uuid.NewString(), t.slug, t.name, t.description, t.defaultTheme, t.sortOrder,
		)
		if err != nil {
			return err
		}
	}

	return nil
}
