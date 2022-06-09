import { Fragment } from 'react'
import { useTable, useExpanded } from 'react-table'

export default function ({ columns, data, renderRowSubComponent, getRowProps = () => ({}), subTable = false }) {
  const { getTableProps, getTableBodyProps, headerGroups, rows, prepareRow, visibleColumns } = useTable({
    columns,
    data
  }, useExpanded)

  return (
    <table className='w-full sticky top-0' {...getTableProps()}>
      <thead>
        {headerGroups.map(headerGroup => (
          <tr key={headerGroup.id} {...headerGroup.getHeaderGroupProps()}>
            {headerGroup.headers.map(column => (
              <th key={column.id} className={`sticky top-0 bg-black z-10 text-left uppercase font-normal text-3xs py-1 text-gray-400 border-b border-gray-800 ${subTable ? 'pb-3' : 'border-b border-gray-800'}`} {...column.getHeaderProps()}>
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
            <Fragment key={row.getRowProps().key}>
              <tr className={`group ${row.isExpanded || (subTable && Number(row.id) === rows.length - 1) ? '' : 'border-b border-gray-800'}  text-2xs`} key={row.id} {...row.getRowProps(getRowProps(row))}>
                {row.cells.map(cell => {
                  return (
                    <td key={cell.id} {...cell.getCellProps()}>
                      {cell.render('Cell')}
                    </td>
                  )
                })}
              </tr>
              {row.isExpanded && (
                <tr>
                  <td colSpan={visibleColumns.length}>
                    {renderRowSubComponent(row)}
                  </td>
                </tr>
              )}
              {row.isExpanded && <tr className='border-b border-gray-800' />}
            </Fragment>
          )
        })}
      </tbody>
    </table>
  )
}
