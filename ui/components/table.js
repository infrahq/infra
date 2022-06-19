import { useTable, useExpanded } from 'react-table'

export default function ({ columns, data, getRowProps = () => {} }) {
  const {
    getTableProps,
    getTableBodyProps,
    prepareRow,
    headerGroups,
    rows
  } = useTable({
    columns,
    data,
    autoResetExpanded: false
  }, useExpanded)

  return (
    <table className='w-full sticky top-0 table-fixed' {...getTableProps()}>
      <thead>
        {headerGroups.map(headerGroup => (
          <tr key={headerGroup.id} {...headerGroup.getHeaderGroupProps()}>
            {headerGroup.headers.map(column => (
              <th width={column.width} key={column.id} className='sticky top-0 bg-black z-10 text-left uppercase font-normal text-3xs py-1 text-gray-400 border-b border-gray-800' {...column.getHeaderProps()}>
                {column.render('Header')}
              </th>
            ))}
          </tr>
        ))}
      </thead>
      <tbody className='relative' {...getTableBodyProps()}>
        {rows.map(row => {
          prepareRow(row)
          const props = row.getRowProps(getRowProps(row))
          return (
            <tr
              {...props}
              key={row.id}
              className={`${props.className} group text-2xs`}
            >
              {row.cells.map(cell => {
                const props = cell.getCellProps()
                return (
                  <td key={cell.id} {...props} className={`${props.className} border-b border-gray-800`}>
                    {cell.render('Cell')}
                  </td>
                )
              })}
            </tr>
          )
        })}
      </tbody>
    </table>
  )
}
