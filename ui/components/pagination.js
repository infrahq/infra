import { useRouter } from 'next/router'
import { ChevronLeftIcon, ChevronRightIcon } from '@heroicons/react/solid'

function getPageNums(selected, count, totalPages) {
  const pageNums = []
  const beginOffset = Math.max(
    Math.floor(count / 2),
    count + selected - totalPages
  )
  for (
    let i = Math.max(1, selected - beginOffset);
    pageNums.length <= count && i <= totalPages;
    i++
  ) {
    pageNums.push(i)
  }

  return pageNums
}

function Pages({ path, selected, count, totalPages }) {
  return getPageNums(selected, count, totalPages).map(page => (
    <a
      href={path + '?p=' + page}
      className={`inline-flex items-center py-2 px-4 text-sm font-medium text-gray-500 hover:text-violet-300 ${
        selected === page ? 'rounded-md bg-gray-700' : ''
      }`}
      key={page}
    >
      {page}
    </a>
  ))
}

function Arrow({ path, direction }) {
  return (
    <a
      href={path}
      className='inline-flex items-center pl-1 text-sm font-medium text-gray-500 hover:text-violet-300 '
    >
      {direction === 'RIGHT' && (
        <ChevronRightIcon
          className='ml-3 h-5 w-5 text-gray-400 hover:text-violet-300'
          aria-hidden='true'
        />
      )}
      {direction === 'LEFT' && (
        <ChevronLeftIcon
          className='ml-3 h-5 w-5 text-gray-400 hover:text-violet-300'
          aria-hidden='true'
        />
      )}
    </a>
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
    totalPages > 1 && (
      <div>
        <nav key='paginator' className='flex justify-end px-4 pb-2'>
          <Arrow direction='LEFT' path={path + '?p=' + (curr > 1 ? curr - 1 : 1)}></Arrow>
          <Pages
            path={path}
            count={7}
            totalPages={totalPages}
            selected={curr}
          ></Pages>
          <Arrow direction='RIGHT' path={path + '?p=' + (curr < totalPages ? curr + 1 : Math.max(1, totalPages))} ></Arrow>
        </nav>
        <h3
          key='results'
          className='flex justify-end px-4 pb-4 text-3xs text-gray-400'
        >
          Displaying {lowerItem}–{upperItem} out of {totalCount}
        </h3>
      </div>
    )
  )
}
