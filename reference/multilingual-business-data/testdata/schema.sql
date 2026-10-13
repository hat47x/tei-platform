-- Reference-only D2 relational projection. Never install into a user's DB.
-- One physical representation for SQLite and PostgreSQL.
CREATE TABLE tei_multilingual_revision (
  tenant_id TEXT NOT NULL,
  record_id TEXT NOT NULL,
  field_scope TEXT NOT NULL,
  field_kind TEXT NOT NULL CHECK (field_kind = 'attribute'),
  field_id TEXT NOT NULL,
  locale TEXT NOT NULL,
  edition TEXT NOT NULL,
  revision TEXT NOT NULL,
  body TEXT NOT NULL CHECK (length(body) > 0),
  origin_kind TEXT NOT NULL CHECK (origin_kind IN ('original', 'translation', 'independent')),
  review_state TEXT NOT NULL CHECK (review_state IN ('candidate', 'approved')),
  source_locale TEXT,
  source_revision TEXT,
  PRIMARY KEY (tenant_id, record_id, field_scope, field_kind, field_id,
               locale, edition, revision),
  CHECK (
    (origin_kind = 'translation' AND source_locale IS NOT NULL
      AND source_revision IS NOT NULL AND source_locale <> locale)
    OR
    (origin_kind <> 'translation' AND source_locale IS NULL AND source_revision IS NULL)
  ),
  FOREIGN KEY (tenant_id, record_id, field_scope, field_kind, field_id,
               source_locale, edition, source_revision)
    REFERENCES tei_multilingual_revision
      (tenant_id, record_id, field_scope, field_kind, field_id,
       locale, edition, revision)
);

CREATE TABLE tei_multilingual_publication (
  tenant_id TEXT NOT NULL,
  record_id TEXT NOT NULL,
  field_scope TEXT NOT NULL,
  field_kind TEXT NOT NULL,
  field_id TEXT NOT NULL,
  locale TEXT NOT NULL,
  edition TEXT NOT NULL,
  revision TEXT NOT NULL,
  PRIMARY KEY (tenant_id, record_id, field_scope, field_kind, field_id,
               locale, edition),
  FOREIGN KEY (tenant_id, record_id, field_scope, field_kind, field_id,
               locale, edition, revision)
    REFERENCES tei_multilingual_revision
      (tenant_id, record_id, field_scope, field_kind, field_id,
       locale, edition, revision)
);

-- Read boundary: neither a candidate nor a stale translation is displayed.
-- No region/language fallback or authority is granted by this view.
CREATE VIEW tei_multilingual_current AS
SELECT r.tenant_id, r.record_id, r.field_scope, r.field_kind, r.field_id,
       r.locale, r.edition, r.revision, r.body, r.origin_kind
FROM tei_multilingual_publication AS p
JOIN tei_multilingual_revision AS r
  ON r.tenant_id = p.tenant_id
 AND r.record_id = p.record_id
 AND r.field_scope = p.field_scope
 AND r.field_kind = p.field_kind
 AND r.field_id = p.field_id
 AND r.locale = p.locale
 AND r.edition = p.edition
 AND r.revision = p.revision
LEFT JOIN tei_multilingual_publication AS sp
  ON sp.tenant_id = r.tenant_id
 AND sp.record_id = r.record_id
 AND sp.field_scope = r.field_scope
 AND sp.field_kind = r.field_kind
 AND sp.field_id = r.field_id
 AND sp.locale = r.source_locale
 AND sp.edition = r.edition
 AND sp.revision = r.source_revision
LEFT JOIN tei_multilingual_revision AS sr
  ON sr.tenant_id = sp.tenant_id
 AND sr.record_id = sp.record_id
 AND sr.field_scope = sp.field_scope
 AND sr.field_kind = sp.field_kind
 AND sr.field_id = sp.field_id
 AND sr.locale = sp.locale
 AND sr.edition = sp.edition
 AND sr.revision = sp.revision
WHERE r.review_state = 'approved'
  AND (r.origin_kind <> 'translation'
       OR (sp.revision IS NOT NULL
           AND sr.review_state = 'approved'
           AND sr.origin_kind <> 'translation'));

CREATE INDEX tei_multilingual_revision_lookup
  ON tei_multilingual_revision(tenant_id, locale, record_id, field_id, edition);
