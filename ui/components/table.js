import { useTable } from 'react-table'

export default function ({ columns, data, getRowProps = () => ({}), highlight = true }) {
  const { getTableProps, getTableBodyProps, headerGroups, rows, prepareRow } = useTable({
    columns,
    data
  })

  return (
    <table className='flex-1 w-full' {...getTableProps()}>
      <thead>
        {headerGroups.map(headerGroup => (
          <tr key={headerGroup.id} {...headerGroup.getHeaderGroupProps()}>
            {headerGroup.headers.map(column => (
              <th key={column.id} className='sticky top-0 bg-black z-10 text-left uppercase font-normal text-3xs py-1 text-gray-400 border-b border-gray-800' {...column.getHeaderProps()}>
                {column.render('Header')}
              </th>
            ))}
          </tr>
        ))}
      </thead>
      <tbody className='relative' {...getTableBodyProps()}>
        {rows.map(row => {
          prepareRow(row)
          return (
            <tr className={`group border-b border-gray-800 text-2xs ${highlight ? 'hover:bg-gray-900/60' : ''}`} key={row.id} {...row.getRowProps(getRowProps(row))}>
              {row.cells.map(cell => {
                return (
                  <td key={cell.id} {...cell.getCellProps()}>
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
