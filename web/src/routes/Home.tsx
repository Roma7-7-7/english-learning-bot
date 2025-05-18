import {type JSX, useEffect, useState} from "react";
import client, {type Words, type WordsQueryParams} from "../api/client.tsx";

interface Pagination {
    page: number;
    pageSize: number;
    totalPages: number;
}

export function Home() {
    const [words, setWords] = useState<Words | null>(null);
    const [qp, setQP] = useState({
        search: "",
        to_review: false,
        offset: 0,
        limit: 15,
    } as WordsQueryParams);
    const [pagination, setPagination] = useState<Pagination>({
        page: 1,
        pageSize: 15,
        totalPages: 1,
    })
    const [error, setError] = useState<string>("");


    useEffect(() => {
        if (error !== "") {
            setError("");
        }

        client.findWords(qp).then(r => {
            if (r.status === 200) {
                return r.json();
            }

            throw new Error("Failed to fetch words");
        }).then(w => {
            if (w) {
                setWords(w);
                setPagination({
                    page: qp.offset / qp.limit + 1,
                    pageSize: qp.limit,
                    totalPages: Math.ceil(w.total / qp.limit),
                });
            } else {
                setError("No words found");
            }
        }).catch(e => {
            console.error("Error fetching words:", e);
            setError("Failed to fetch words");
            setWords(null);
        })
    }, [qp])

    const onPageChange = (page: number) => {
        if (error !== "") {
            setError("");
        }
        setQP({
            ...qp,
            offset: (page - 1) * qp.limit,
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
                            <label><input className="form-control" type="text" name="search" placeholder="Search" value={qp.search} onChange={present => {
                                setQP({
                                    ...qp,
                                    search: present.target.value,
                                    offset: 0,
                                });
                            }}/></label>
                        </div>
                        <div className="col-2">
                            <div className="form-check d-flex align-items-center h-100">
                                <label className="form-check-label ms-2">
                                    <input name="to_review" type="checkbox" className="form-check-input" checked={qp.to_review} onChange={present => {
                                        setQP({
                                            ...qp,
                                            to_review: present.target.checked,
                                        });
                                    }}/> To Review
                                </label>
                            </div>
                        </div>
                        <div className="col-3"></div>
                        <div
                            className="col-3">
                            <button className="btn btn-primary" style={{width: '100%'}}>Submit</button>
                        </div>
                        <div className="col-1">
                            <a className="btn btn-secondary" onClick={() => {
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
                                    {words.items.map((word) => {
                                        return (
                                            <tr key={word.word}>
                                                <td>{word.word}</td>
                                                <td>{word.translation}</td>
                                                <td>{word.to_review ? "Yes" : "No"}</td>
                                                <td className="text-center">
                                                    <a href={`/words/edit/${word.word}`} className="btn btn-link bi bi-pencil"></a>
                                                </td>
                                                <td className="text-center">
                                                    <button className="btn btn-link bi bi-trash" data-word={word.word}></button>
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
                                    {paginationFooter(pagination, onPageChange)}
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

function paginationFooter(pagination: Pagination, onPageChange: (page: number) => void): JSX.Element {
    if (pagination.totalPages <= 1) {
        return (<></>)
    }

    interface li {
        active: boolean;
        disabled: boolean;
        page: number;
        isArrow?: boolean;
    }

    const items: li[] = [];

    if (pagination.totalPages <= 7) {
        for (let i = 1; i <= pagination.totalPages; i++) {
            items.push({
                active: i === pagination.page,
                disabled: false,
                page: i
            });
        }
    } else {
        items.push({
            active: false,
            disabled: pagination.page === 1,
            page: 1,
            isArrow: true
        });

        // &nbsp;&nbsp;

        if (pagination.page > 2) {
            items.push({
                active: false,
                disabled: false,
                page: pagination.page - 2
            });
        }

        if (pagination.page > 1) {
            items.push({
                active: false,
                disabled: false,
                page: pagination.page - 1
            });
        }

        items.push({
            active: true,
            disabled: true,
            page: pagination.page
        });

        if (pagination.page < pagination.totalPages) {
            items.push({
                active: false,
                disabled: false,
                page: pagination.page + 1
            });
        }

        if (pagination.page < pagination.totalPages - 1) {
            items.push({
                active: false,
                disabled: false,
                page: pagination.page + 2
            });
        }

        // &nbsp;&nbsp;

        items.push({
            active: false,
            disabled: pagination.page === pagination.totalPages,
            page: pagination.totalPages,
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