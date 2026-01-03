-- ============================================================================
-- Seed: Master Skills Data
-- Purpose: Populate skills table with proper categorization
-- ============================================================================

-- Clear existing skills (if any) to avoid duplicates
TRUNCATE TABLE skills CASCADE;

-- ============================================================================
-- COMPUTER SKILLS
-- ============================================================================
INSERT INTO skills (name, category) VALUES
('Microsoft Office (Word, Excel, PowerPoint)', 'COMPUTER'),
('AutoCAD', 'COMPUTER'),
('SAP', 'COMPUTER'),
('Adobe Photoshop', 'COMPUTER'),
('Adobe Illustrator', 'COMPUTER'),
('Programming (Python, Java, etc.)', 'COMPUTER'),
('Web Development (HTML, CSS, JS)', 'COMPUTER'),
('Database Management (SQL)', 'COMPUTER'),
('ERP Systems', 'COMPUTER'),
('CNC Programming', 'COMPUTER'),
('PLC Programming', 'COMPUTER'),
('3D Modeling (SolidWorks, Fusion)', 'COMPUTER'),
('Data Analysis (Power BI, Tableau)', 'COMPUTER'),
('Video Editing (Premiere, Final Cut)', 'COMPUTER');

-- ============================================================================
-- LANGUAGE SKILLS
-- ============================================================================
INSERT INTO skills (name, category) VALUES
('Japanese (日本語)', 'LANGUAGE'),
('English', 'LANGUAGE'),
('Mandarin (中文)', 'LANGUAGE'),
('Korean (한국어)', 'LANGUAGE'),
('Indonesian (Bahasa)', 'LANGUAGE'),
('Vietnamese', 'LANGUAGE'),
('Thai', 'LANGUAGE');

-- ============================================================================
-- SOFT SKILLS
-- ============================================================================
INSERT INTO skills (name, category) VALUES
('Leadership', 'SOFT'),
('Communication', 'SOFT'),
('Teamwork', 'SOFT'),
('Problem Solving', 'SOFT'),
('Time Management', 'SOFT'),
('Adaptability', 'SOFT'),
('Critical Thinking', 'SOFT'),
('Conflict Resolution', 'SOFT'),
('Customer Service', 'SOFT'),
('Presentation Skills', 'SOFT');

-- ============================================================================
-- TECHNICAL / INDUSTRY SKILLS
-- ============================================================================
INSERT INTO skills (name, category) VALUES
('Welding (溶接)', 'TECHNICAL'),
('Machining/Lathe Operation', 'TECHNICAL'),
('Quality Control (QC/QA)', 'TECHNICAL'),
('Forklift Operation', 'TECHNICAL'),
('Crane Operation', 'TECHNICAL'),
('Electrical Work', 'TECHNICAL'),
('Plumbing', 'TECHNICAL'),
('HVAC', 'TECHNICAL'),
('Automotive Repair', 'TECHNICAL'),
('Food Processing', 'TECHNICAL'),
('Care Worker (介護)', 'TECHNICAL'),
('Nursing', 'TECHNICAL'),
('Construction', 'TECHNICAL'),
('Agriculture', 'TECHNICAL'),
('Hospitality', 'TECHNICAL');
