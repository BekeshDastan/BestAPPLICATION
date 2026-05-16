/* CSS-columns masonry — no layout library needed */
export default function MasonryGrid({ items, renderItem, cols = 3, gap = 8 }) {
  if (!items.length) return null
  return (
    <div
      style={{
        columns:   cols,
        columnGap: gap,
      }}
    >
      {items.map((item, i) => (
        <div
          key={item.id ?? i}
          style={{ breakInside: 'avoid', marginBottom: gap }}
        >
          {renderItem(item, i)}
        </div>
      ))}
    </div>
  )
}

/* Responsive wrapper: 3 cols on lg+, 2 on sm+ */
export function ResponsiveMasonry({ items, renderItem, gap = 8 }) {
  return (
    <>
      {/* Desktop: 3 cols */}
      <div className="hidden lg:block">
        <MasonryGrid items={items} renderItem={renderItem} cols={3} gap={gap} />
      </div>
      {/* Mobile/tablet: 2 cols */}
      <div className="lg:hidden">
        <MasonryGrid items={items} renderItem={renderItem} cols={2} gap={gap} />
      </div>
    </>
  )
}
