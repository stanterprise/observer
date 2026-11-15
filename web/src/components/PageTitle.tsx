type PageTitleProps = {
  title: string;
};

export function PageTitle({ title }: PageTitleProps) {
  const pageTitle = title.trim() ? `${title.trim()} · Observer` : "Observer";
  return <title>{pageTitle}</title>;
}
