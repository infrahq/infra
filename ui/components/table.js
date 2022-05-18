import { useTable } from 'react-table'

export default function ({ columns, data, getRowProps = () => ({}) }) {
  const { getTableProps, getTableBodyProps, headerGroups, rows, prepareRow } = useTable({
    columns,
    data
  })

  return (
    <table className='w-full table-auto' {...getTableProps()}>
      <thead>
        {headerGroups.map(headerGroup => (
          <tr key={headerGroup.id} {...headerGroup.getHeaderGroupProps()}>
            {headerGroup.headers.map(column => (
              <th key={column.id} className='text-left uppercase font-normal text-[11px] py-1 text-gray-400 border-b border-zinc-800' {...column.getHeaderProps()}>
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
            <tr className='group border-b border-zinc-800' key={row.id} {...row.getRowProps(getRowProps(row))}>
              {row.cells.map(cell => {
                return (
                  <td className='group-hover:bg-zinc-900 text-sm py-[3px]' key={cell.id} {...cell.getCellProps()}>
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
