import { useRouter } from 'next/router'
import Link from 'next/link'
import { ChevronLeftIcon, ChevronRightIcon } from '@heroicons/react/solid'

function Pages({ path, selected, count, totalPages }) {
  const pages = []
  const beginOffset = Math.max(
    Math.floor(count / 2),
    count - 1 + selected - totalPages
  )
  for (
    let page = Math.max(1, selected - beginOffset);
    pages.length < count && page <= totalPages;
    page++
  ) {
    pages.push(<Link key={page} href={path + '?p=' + page}>
    <a
      className={`inline-flex w-8 items-center px-1 text-center text-sm font-medium text-gray-500 hover:text-violet-300 ${
        selected === page ? 'rounded-md text-violet-300' : ''
      }`}
    >
      {page}
    </a>
  </Link>)
  }

  return pages.map(page => (
    page
  ))
}

function Arrow({ path, direction }) {
  return (
    <Link href={path}>
      <a className='inline-flex items-center text-sm font-medium text-gray-500 hover:text-violet-300 '>
        {direction === 'RIGHT' && (
          <ChevronRightIcon
            className='h-5 w-5 text-gray-400 hover:text-violet-300'
            aria-hidden='true'
          />
        )}
        {direction === 'LEFT' && (
          <ChevronLeftIcon
            className='mr-3 h-5 w-5 text-gray-400 hover:text-violet-300'
            aria-hidden='true'
          />
        )}
      </a>
    </Link>
  )
}

export default function Pagination({
  curr = 1,
  totalPages = 0,
  totalCount = 0,
}) {
  const router = useRouter()
  const path = router.pathname
  const limit = 13 // TODO: dynamic limit

  curr = parseInt(curr)
  totalPages = parseInt(totalPages)
  totalCount = parseInt(totalCount)

  const lowerItem = totalCount === 0 ? 0 : 1 + (curr - 1) * limit
  const upperItem = Math.min(1 + (curr - 1) * limit + limit, totalCount)

  return (
    <div className='box-border flex items-center justify-between pb-[16.5px]'>
      <h3 key='results' className='px-4 pb-2 text-2xs text-gray-400'>
        Displaying {lowerItem}–{upperItem} out of {totalCount}
      </h3>
      <div className='flex items-center px-4 pb-2'>
        <Arrow
          direction='LEFT'
          path={path + '?p=' + (curr > 1 ? curr - 1 : 1)}
        ></Arrow>
        <Pages
          path={path}
          count={7}
          totalPages={totalPages}
          selected={curr}
        ></Pages>
        <Arrow
          direction='RIGHT'
          path={
            path +
            '?p=' +
            (curr < totalPages ? curr + 1 : Math.max(1, totalPages))
          }
        ></Arrow>
      </div>
    </div>
  )
}
