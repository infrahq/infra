import React, { useEffect, useContext } from 'react'

export const BreadcrumbsContext = React.createContext([])

export default function Breadcrumbs({ children }) {
  const [, setBreadcrumbs] = useContext(BreadcrumbsContext)

  // update components
  useEffect(() => {
    // on mount set context
    setBreadcrumbs(children)

    // on unmount reset to []
    return () => {
      setBreadcrumbs([])
    }
  }, [children, setBreadcrumbs])
}
