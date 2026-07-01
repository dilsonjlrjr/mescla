-- Paint Match AI — Initial Database Schema
-- paint_knowledge.db

PRAGMA foreign_keys = ON;
PRAGMA journal_mode = WAL;

-- ============================================
-- LOOKUP TABLES
-- ============================================

CREATE TABLE IF NOT EXISTS paint_types (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL UNIQUE
);

CREATE TABLE IF NOT EXISTS finish_types (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL UNIQUE
);

CREATE TABLE IF NOT EXISTS coverage_types (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL UNIQUE
);

CREATE TABLE IF NOT EXISTS opacity_types (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL UNIQUE
);

-- ============================================
-- CORE TABLES
-- ============================================

CREATE TABLE IF NOT EXISTS manufacturers (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL UNIQUE,
    country TEXT,
    website TEXT,
    logo_path TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS product_lines (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    manufacturer_id INTEGER NOT NULL,
    name TEXT NOT NULL,
    description TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (manufacturer_id) REFERENCES manufacturers(id),
    -- Sem esta UNIQUE o "ON CONFLICT DO NOTHING" do seed nunca dispara e
    -- reimportar o catálogo duplica linhas de produto (e, em cascata, tintas).
    UNIQUE(manufacturer_id, name)
);

CREATE INDEX IF NOT EXISTS idx_product_lines_manufacturer ON product_lines(manufacturer_id);

CREATE TABLE IF NOT EXISTS paints (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    manufacturer_id INTEGER NOT NULL,
    product_line_id INTEGER NOT NULL,
    code TEXT NOT NULL,
    name TEXT NOT NULL,
    paint_type_id INTEGER,
    finish_type_id INTEGER,
    coverage_type_id INTEGER,
    opacity_type_id INTEGER,
    volume_ml REAL,
    thumbnail_path TEXT,
    image_path TEXT,
    notes TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (manufacturer_id) REFERENCES manufacturers(id),
    FOREIGN KEY (product_line_id) REFERENCES product_lines(id),
    FOREIGN KEY (paint_type_id) REFERENCES paint_types(id),
    FOREIGN KEY (finish_type_id) REFERENCES finish_types(id),
    FOREIGN KEY (coverage_type_id) REFERENCES coverage_types(id),
    FOREIGN KEY (opacity_type_id) REFERENCES opacity_types(id),
    UNIQUE(manufacturer_id, product_line_id, code)
);

CREATE INDEX IF NOT EXISTS idx_paints_manufacturer ON paints(manufacturer_id);
CREATE INDEX IF NOT EXISTS idx_paints_line ON paints(product_line_id);
CREATE INDEX IF NOT EXISTS idx_paints_code ON paints(code);
CREATE INDEX IF NOT EXISTS idx_paints_name ON paints(name);

-- ============================================
-- COLOR DATA
-- ============================================

CREATE TABLE IF NOT EXISTS paint_colors (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    paint_id INTEGER NOT NULL UNIQUE,
    rgb_r INTEGER NOT NULL,
    rgb_g INTEGER NOT NULL,
    rgb_b INTEGER NOT NULL,
    hsv_h REAL NOT NULL,
    hsv_s REAL NOT NULL,
    hsv_v REAL NOT NULL,
    hsl_h REAL NOT NULL,
    hsl_s REAL NOT NULL,
    hsl_l REAL NOT NULL,
    lab_l REAL NOT NULL,
    lab_a REAL NOT NULL,
    lab_b REAL NOT NULL,
    lch_l REAL NOT NULL,
    lch_c REAL NOT NULL,
    lch_h REAL NOT NULL,
    swatch_path TEXT,
    delta_e_method TEXT DEFAULT '2000',
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (paint_id) REFERENCES paints(id)
);

CREATE INDEX IF NOT EXISTS idx_paint_colors_paint ON paint_colors(paint_id);

-- ============================================
-- EQUIVALENCES
-- ============================================

CREATE TABLE IF NOT EXISTS equivalences (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    source_paint_id INTEGER NOT NULL,
    target_paint_id INTEGER NOT NULL,
    similarity REAL NOT NULL,
    delta_e REAL NOT NULL,
    delta_e_method TEXT NOT NULL DEFAULT '2000',
    source TEXT,
    notes TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (source_paint_id) REFERENCES paints(id),
    FOREIGN KEY (target_paint_id) REFERENCES paints(id),
    UNIQUE(source_paint_id, target_paint_id)
);

CREATE INDEX IF NOT EXISTS idx_equivalences_source ON equivalences(source_paint_id);
CREATE INDEX IF NOT EXISTS idx_equivalences_target ON equivalences(target_paint_id);

-- ============================================
-- RECIPES
-- ============================================

CREATE TABLE IF NOT EXISTS recipes (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    target_paint_id INTEGER NOT NULL,
    source_manufacturer_id INTEGER,
    target_manufacturer_id INTEGER,
    precision_score REAL,
    delta_e REAL,
    method TEXT,
    source TEXT,
    notes TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (target_paint_id) REFERENCES paints(id),
    FOREIGN KEY (source_manufacturer_id) REFERENCES manufacturers(id),
    FOREIGN KEY (target_manufacturer_id) REFERENCES manufacturers(id)
);

CREATE INDEX IF NOT EXISTS idx_recipes_target ON recipes(target_paint_id);
CREATE INDEX IF NOT EXISTS idx_recipes_source_manufacturer ON recipes(source_manufacturer_id);
CREATE INDEX IF NOT EXISTS idx_recipes_target_manufacturer ON recipes(target_manufacturer_id);

CREATE TABLE IF NOT EXISTS recipe_ingredients (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    recipe_id INTEGER NOT NULL,
    paint_id INTEGER NOT NULL,
    percentage REAL NOT NULL,
    sort_order INTEGER NOT NULL DEFAULT 0,
    FOREIGN KEY (recipe_id) REFERENCES recipes(id),
    FOREIGN KEY (paint_id) REFERENCES paints(id)
);

CREATE INDEX IF NOT EXISTS idx_recipe_ingredients_recipe ON recipe_ingredients(recipe_id);
CREATE INDEX IF NOT EXISTS idx_recipe_ingredients_paint ON recipe_ingredients(paint_id);

-- ============================================
-- RESOURCES
-- ============================================

CREATE TABLE IF NOT EXISTS resources (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    paint_id INTEGER,
    manufacturer_id INTEGER,
    type TEXT NOT NULL,
    path TEXT,
    url TEXT,
    description TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (paint_id) REFERENCES paints(id),
    FOREIGN KEY (manufacturer_id) REFERENCES manufacturers(id)
);

CREATE INDEX IF NOT EXISTS idx_resources_paint ON resources(paint_id);
CREATE INDEX IF NOT EXISTS idx_resources_manufacturer ON resources(manufacturer_id);
CREATE INDEX IF NOT EXISTS idx_resources_type ON resources(type);
