import {type JSX, useEffect, useState} from "react";
import client, {type Words, type WordsQueryParams} from "../api/client.tsx";
import {useAppState} from "../context.tsx";

export function Home() {
    const {refreshStats} = useAppState()
    const [words, setWords] = useState<Words | null>(null);
    const [qp, setQP] = useState({
        search: "",
        to_review: false,
        offset: 0,
        limit: 15,
    } as WordsQueryParams);
    const [error, setError] = useState<string>("");

    function fetchWords() {
        if (error !== "") {
            setError("");
        }

        client.findWords(qp).then(r => {
            if (r.status === 200) {
                return r.json() as Promise<Words>;
            }

            throw new Error("Failed to fetch words");
        }).then(w => {
            console.log("Words:", w.items.length, w.total);
            // It may happen if we applied some filtering which has words but current page overflows the total number of filtered words
            if (w.items.length == 0 && w.total > 0) {
                setQP(existing => {
                    return {
                        ...existing,
                        offset: 0,
                    }
                })
                return
            }

            setWords(w);
        }).catch(e => {
            console.error("Error fetching words:", e);
            setError("Failed to fetch words");
            setWords(null);
        })
    }

    useEffect(() => {
        fetchWords()
    }, [qp])

    function handleDeleteWord(word: string) {
        if (confirm(`Are you sure you want to delete the word "${word}"?`)) {
            client.deleteWord(word).then(r => {
                if (r.status === 200) {
                    refreshStats()
                    fetchWords()
                } else {
                    setError("Failed to delete word");
                }
            }).catch(e => {
                console.error("Error deleting word:", e);
                setError("Failed to delete word");
            })
        }
    }

    const onPageChange = (page: number) => {
        setQP((existing: WordsQueryParams) => {
            return {
                ...existing,
                offset: (page - 1) * existing.limit,
            }
        });
    }

    return (
        <>
            {!words && (
                <div><h1>Loading...</h1></div>
            )}
            {words && (
                <div id="content" className="p-3">
                    <form id="searchForm" className="row form-inline form-group align-items-center mb-3">
                        <div className="col-3">
                            <label><input className="form-control" type="text" name="search" placeholder="Search" value={qp.search}
                                          onChange={present => {
                                              setQP((existing: WordsQueryParams) => {
                                                  return {
                                                      ...existing,
                                                      search: present.target.value,
                                                  }
                                              });
                                          }}/></label>
                        </div>
                        <div className="col-2">
                            <div className="form-check d-flex align-items-center h-100">
                                <label className="form-check-label ms-2">
                                    <input name="to_review" type="checkbox" className="form-check-input" checked={qp.to_review}
                                           onChange={present => {
                                               setQP((existing: WordsQueryParams) => {
                                                   return {
                                                       ...existing,
                                                       to_review: present.target.checked,
                                                   }
                                               });
                                           }}/> To Review
                                </label>
                            </div>
                        </div>
                        <div className="col-6"></div>
                        <div className="col-1">
                            <a className="btn btn-secondary" style={{width: '100%'}} onClick={() => {
                                setQP({
                                    search: "",
                                    to_review: false,
                                    offset: 0,
                                    limit: 15,
                                });
                            }}><span aria-hidden="true">&times;</span></a>
                        </div>
                    </form>
                    <div id="words">
                        <div className="row">
                            <div className="col-12">
                                <table className="table">
                                    <thead>
                                    <tr>
                                        <th>Word</th>
                                        <th>Translation</th>
                                        <th>To Review</th>
                                        <th>Edit</th>
                                        <th>Delete</th>
                                    </tr>
                                    </thead>
                                    <tbody>
                                    {words.items.map((item) => {
                                        return (
                                            <tr key={item.word}>
                                                <td>{item.word}</td>
                                                <td>{item.translation}</td>
                                                <td>{item.to_review ? "Yes" : "No"}</td>
                                                <td className="text-center">
                                                    <a href={`/words/edit/${item.word}`} className="btn btn-link bi bi-pencil"></a>
                                                </td>
                                                <td className="text-center">
                                                    <button className="btn btn-link bi bi-trash"
                                                            onClick={() => handleDeleteWord(item.word)}></button>
                                                </td>
                                            </tr>
                                        )
                                    })}
                                    </tbody>
                                </table>
                            </div>
                        </div>
                        <div className="row">
                            <div className="col-12">
                                <div className="d-flex justify-content-center">
                                    {paginationFooter(qp, words.total, onPageChange)}
                                </div>
                            </div>
                        </div>
                    </div>
                    <div className="row">
                        {error && (
                            <div className="alert alert-danger" role="alert">
                                {error}
                            </div>
                        )}
                    </div>
                </div>
            )}
        </>
    )
}

function paginationFooter(qp: WordsQueryParams, totalItems: number, onPageChange: (page: number) => void): JSX.Element {
    const totalPages = Math.ceil(totalItems / qp.limit);
    const page = getPage(qp)
    if (totalPages <= 1) {
        return (<></>)
    }


    interface li {
        active: boolean;
        disabled: boolean;
        page: number;
        isArrow?: boolean;
    }

    const items: li[] = [];

    if (totalPages <= 7) {
        for (let i = 1; i <= totalPages; i++) {
            items.push({
                active: i === page,
                disabled: false,
                page: i
            });
        }
    } else {
        items.push({
            active: false,
            disabled: page === 1,
            page: 1,
            isArrow: true
        });

        // &nbsp;&nbsp;

        if (page > 2) {
            items.push({
                active: false,
                disabled: false,
                page: page - 2
            });
        }

        if (page > 1) {
            items.push({
                active: false,
                disabled: false,
                page: page - 1
            });
        }

        items.push({
            active: true,
            disabled: true,
            page: page
        });

        if (page < totalPages) {
            items.push({
                active: false,
                disabled: false,
                page: page + 1
            });
        }

        if (page < totalPages - 1) {
            items.push({
                active: false,
                disabled: false,
                page: page + 2
            });
        }

        // &nbsp;&nbsp;

        items.push({
            active: false,
            disabled: page === totalPages,
            page: totalPages,
            isArrow: true
        });
    }

    return <ul className="pagination">
        {items.map(((item, idx) => {
            return (
                <li key={"page-" + idx} className={`page-item ${item.active ? "active" : ""} ${item.disabled ? "disabled" : ""}`}>
                    <a className="page-link" onClick={() => {
                        if (!item.disabled) {
                            onPageChange(item.page);
                        }
                    }}>
                        {item.isArrow && idx === 0 && <span aria-hidden="true">&laquo;</span>}
                        {!item.isArrow && item.page}
                        {item.isArrow && idx === items.length - 1 && <span aria-hidden="true">&raquo;</span>}
                    </a>
                </li>
            )
        }))}
    </ul>
}

function getPage(qp: WordsQueryParams): number {
    return Math.floor(qp.offset / qp.limit) + 1;
}