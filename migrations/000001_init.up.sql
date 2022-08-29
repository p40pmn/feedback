CREATE TABLE questions (
  id varchar(8) NOT NULL PRIMARY KEY,
  title text NOT NULL,
  is_display bool NOT NULL DEFAULT TRUE,
  updated_by varchar(100) NOT NULL,
  updated_at timestamptz NOT NULL
);

CREATE TABLE feedback_remarks(
  id varchar(8) NOT NULL PRIMARY KEY,
  teaching_id varchar(8) NOT NULL,
  question_id varchar(8) NOT NULL,
  rating decimal NOT NULL
);
