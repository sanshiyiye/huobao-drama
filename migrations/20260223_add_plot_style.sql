-- Add plot_style column to dramas table
ALTER TABLE dramas ADD COLUMN plot_style VARCHAR(50);

-- Set default value for existing records
UPDATE dramas SET plot_style = '' WHERE plot_style IS NULL;
